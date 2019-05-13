package redis

import (
	"container/list"
	"errors"
	"net"
	"sync"
	"time"
)

type Pool struct {
	// tcp connection
	Dial func() (Conn, error)

	// ping test before real action
	TestPing func(c Conn, t time.Time) error

	// Max idle connections
	MaxIdle int

	// Max active connections
	MaxActive int

	// idle connection lifetime, if connected time of connection > idletimeout
	// then passive close
	IdleTimeout time.Duration

	// true means wait the pool get new connection
	Wait bool

	// mu protects
	mu sync.Mutex

	// for wait
	cond *sync.Cond

	// pool status
	close bool

	// pool active connections
	active int

	// store idle connections type idleConn
	// data structure is a doubly linked list.
	idleList list.List
}

type idleConn struct {
	// connection
	c Conn
	// the time of put into list
	t time.Time
}

func (p *Pool) Get() Conn {
	c, err := p.get()
	if err != nil {
		return errorConnection{err}
	}

	return &pooledConn{
		p: p,
		c: c,
	}
}

func (p *Pool) ActiveCount() int {
	p.mu.Lock()
	activeCount := p.active
	p.mu.Unlock()

	return activeCount
}

func (p *Pool) Close() error {
	p.mu.Lock()
	idle := p.idleList
	p.idleList.Init() //clear all
	p.close = true
	p.active -= idle.Len()
	if p.cond != nil {
		p.cond.Broadcast()
	}
	p.mu.Unlock()

	// 关闭所有connection
	for i := idle.Front(); i != nil; i = i.Next() {
		i.Value.(idleConn).c.Close()
	}

	return nil
}

func (p *Pool) get() (Conn, error) {
	p.mu.Lock()
	// close timeout idle connection
	if timeout := p.IdleTimeout; timeout > 0 {
		for i, n := 0, p.idleList.Len(); i < n; i++ {
			idleNode := p.idleList.Back()
			if idleNode == nil {
				break
			}
			ic := idleNode.Value.(idleConn)
			// idle连接的时间+idle超时时间 > 当前时间
			// 说明idle连接还可以用，因为取得是双向链表的最后一个，也就是最晚的一个
			// 所以直接break，idle timeout 检查结束
			if ic.t.Add(timeout).After(time.Now()) {
				break
			}
			// 如果不满足上述条件，说明idle connection已过期，需要进行处理
			// 从list中删除
			p.idleList.Remove(idleNode)
			// active - 1, 通知其他请求conn
			p.release()

			p.mu.Unlock()
			ic.c.Close() // 关闭tcp连接
			p.mu.Lock()
		}
	}

	for {
		// Get idle connection
		for i, n := 0, p.idleList.Len(); i < n; i++ {
			idleNode := p.idleList.Front()
			if idleNode == nil {
				break
			}
			ic := idleNode.Value.(idleConn)
			// 从list中删除
			p.idleList.Remove(idleNode)
			p.mu.Unlock()

			// 是否可以ping通
			testFunc := p.TestPing
			if testFunc == nil || testFunc(ic.c, ic.t) == nil {
				return ic.c, nil
			}
			// 不可以ping通，关闭tcp连接
			ic.c.Close()

			p.mu.Lock()
			// active - 1, 通知其他请求conn
			p.release()
		}

		if p.close {
			p.mu.Unlock()
			return nil, errors.New("redispoolclient by go: pool is closed")
		}

		// Dial new connection if under limit
		if p.MaxActive == 0 || p.active < p.MaxActive {
			dial := p.Dial
			p.active += 1
			p.mu.Unlock()
			c, err := dial()
			if err != nil {
				p.mu.Lock()
				p.release()
				p.mu.Unlock()
				c = nil
			}
			return c, err
		}

		// 是否要等待
		if !p.Wait {
			p.mu.Unlock()
			return nil, errors.New("redispoolclient by go: pool is at max count")
		}

		// 通过cond等待
		if p.cond == nil {
			p.cond = sync.NewCond(&p.mu)
		}
		p.cond.Wait()
	}
}

func (p *Pool) release() {
	p.active -= 1
	if p.cond != nil {
		// 通知等待需要连接的请求
		p.cond.Signal()
	}
}

func (p *Pool) put(c Conn, forceClose bool) error {
	err := c.Err()
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.close && err == nil && !forceClose {
		p.idleList.PushFront(idleConn{
			c: c,
			t: time.Now(),
		})
		if p.idleList.Len() > p.MaxIdle {
			c = p.idleList.Remove(p.idleList.Back()).(idleConn).c
		} else {
			c = nil
		}
	}

	if c == nil {
		if p.cond != nil {
			p.cond.Signal()
		}
		return nil
	}

	// 连接过多，关闭一个
	p.release()
	return c.Close()
}

type pooledConn struct {
	p *Pool
	c Conn
}

func (pc *pooledConn) Close() error {
	c := pc.c
	if _, ok := c.(errorConnection); ok {
		return nil
	}
	pc.c = errorConnection{
		err: errors.New("connection is closed"),
	}
	return pc.p.put(c, false)
}

// err
func (pc *pooledConn) Err() error {
	return pc.c.Err()
}

// execute redis command
func (pc *pooledConn) CMD(cmd string, args ...interface{}) (interface{}, error) {
	return pc.c.CMD(cmd, args...)
}

// send cmd to redis server
func (pc *pooledConn) Send(cmd string, args ...interface{}) error {
	return pc.c.Send(cmd, args...)
}

// receive cmd from redis server
func (pc *pooledConn) Receive() (interface{}, error) {
	return pc.c.Receive()
}

// redis server addr
func (pc *pooledConn) RemoteAddr() net.Addr {
	return pc.c.RemoteAddr()
}

// fake one
type errorConnection struct {
	err error
}

func (ec errorConnection) CMD(string, ...interface{}) (interface{}, error) {
	return nil, ec.err
}

func (ec errorConnection) Send(string, ...interface{}) error {
	return ec.err
}

func (ec errorConnection) Err() error {
	return ec.err
}

func (ec errorConnection) Close() error {
	return ec.err
}

func (ec errorConnection) Receive() (interface{}, error) {
	return nil, ec.err
}

func (ec errorConnection) RemoteAddr() net.Addr {
	return nil
}
