package redis

import (
	"sync"
	"testing"
	"time"
)

type poolTestConn struct {
	d   *poolDialer
	err error
	// 嵌入
	Conn
}

// open数目减1后，再去关闭tcp连接
func (c *poolTestConn) Close() error {
	c.d.mu.Lock()
	c.d.open -= 1
	c.d.mu.Unlock()

	return c.Conn.Close()
}

type poolDialer struct {
	mu       sync.Mutex
	t        *testing.T
	dialed   int
	open     int
	commands []string
	dialErr  error
}

func (d *poolDialer) dial() (Conn, error) {
	d.mu.Lock()
	d.dialed += 1
	dialErr := d.dialErr
	d.mu.Unlock()
	if dialErr != nil {
		return nil, dialErr
	}

	c, err := dialDefaultServer()
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.open += 1
	d.mu.Unlock()

	return &poolTestConn{
		d:    d,
		Conn: c,
	}, nil
}

func (d *poolDialer) check(message string, p *Pool, dialed, open int) {
	d.mu.Lock()
	if d.dialed != dialed {
		d.t.Errorf("msg:%s dialed=%d want %d", message, d.dialed, dialed)
	}
	if d.open != open {
		d.t.Errorf("msg:%s open=%d want %d", message, d.open, open)
	}
	if p.ActiveCount() != open {
		d.t.Errorf("msg:%s active=%d want %d", message, p.ActiveCount(), open)
	}
	d.mu.Unlock()
}

func TestPoolReuse(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle: 2,
		Dial:    d.dial,
	}

	for i := 0; i < 10; i++ {
		c1 := p.Get()
		c1.CMD("PING")
		c2 := p.Get()
		c2.CMD("PING")
		c1.Close()
		c2.Close()
	}
	d.check("before close", p, 2, 2)
	p.Close()
	d.check("after close", p, 2, 0)
}

func TestMaxIdle(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle: 2,
		Dial:    d.dial,
	}

	for i := 0; i < 10; i++ {
		c1 := p.Get()
		c1.CMD("PING")
		c2 := p.Get()
		c2.CMD("PING")
		// idle exhausted create new one
		c3 := p.Get()
		c3.CMD("PING")
		c1.Close()
		c2.Close()
		c3.Close()
	}
	// 12 = 2(pool) + 10(new conn)
	d.check("before close", p, 12, 2)
	p.Close()
	d.check("after close", p, 12, 0)
}

func TestPoolClose(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle: 2,
		Dial:    d.dial,
	}
	defer p.Close()

	c1 := p.Get()
	c1.CMD("PING")
	c2 := p.Get()
	c2.CMD("PING")
	// c3 not in idle list
	c3 := p.Get()
	c3.CMD("PING")

	c1.Close()
	if _, err := c1.CMD("PING"); err == nil {
		t.Errorf("need c1 return err")
	}

	// put in pool
	c2.Close()
	c2.Close()

	// real close
	p.Close()

	// c3 connection应该还在
	d.check("after pool close", p, 3, 1)

	if _, err := c1.CMD("PING"); err == nil {
		t.Errorf("need c1 return err")
	}

	c3.Close()

	// c3 关闭
	d.check("after pool close", p, 3, 0)

	if _, err := c1.CMD("PING"); err == nil {
		t.Errorf("need c1 return err")
	}
}

func TestPoolTimeout(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle:     2,
		IdleTimeout: 100 * time.Millisecond,
		Dial:        d.dial,
	}
	defer p.Close()

	c1 := p.Get()
	c1.CMD("PING")
	c2 := p.Get()
	c2.CMD("PING")
	c1.Close()
	c2.Close()

	d.check("start now ", p, 2, 2)

	time.Sleep(p.IdleTimeout)
	c3 := p.Get()
	c3.CMD("PING")
	c3.Close()
	d.check("end now ", p, 3, 1)

}

func TestPoolMaxActive(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle:   2,
		MaxActive: 2,
		Dial:      d.dial,
	}
	defer p.Close()

	c1 := p.Get()
	c1.CMD("PING")

	c2 := p.Get()
	c2.CMD("PING")

	d.check("1", p, 2, 2)
	c3 := p.Get()
	if _, err := c3.CMD("PING"); err == nil {
		t.Errorf("expected err")
	} else {
		//t.Logf("exhausted err info %s", err)
	}

	c3.Close()
	d.check("2", p, 2, 2)

	c2.Close()
	d.check("3", p, 2, 2)

	c3 = p.Get()
	if _, err := c3.CMD("PING"); err != nil {
		t.Errorf("expected nil")
	}
	c3.Close()

	d.check("4", p, 2, 2)
}

func startGoroutines(p *Pool, cmd string, args ...interface{}) chan error {
	errs := make(chan error, 10)
	for i := 0; i < cap(errs); i++ {
		go func() {
			c := p.Get()
			_, err := c.CMD(cmd)
			errs <- err
			c.Close()
		}()
	}

	time.Sleep(time.Second)
	return errs
}

func TestWaitPool(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle:   1,
		MaxActive: 1,
		Dial:      d.dial,
		Wait:      true,
	}
	defer p.Close()

	c := p.Get()
	errs := startGoroutines(p, "PING")
	d.check("befor close", p, 1, 1)
	c.Close()

	timeout := time.After(2 * time.Second)
	for i := 0; i < cap(errs); i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatal(err)
			}
		case <-timeout:
			t.Fatal("timeout")
		}
	}
	d.check("done", p, 1, 1)
}

func TestWaitPoolClose(t *testing.T) {
	d := poolDialer{
		t: t,
	}
	p := &Pool{
		MaxIdle:   1,
		MaxActive: 1,
		Dial:      d.dial,
		Wait:      true,
	}
	defer p.Close()

	c := p.Get()
	if _, err := c.CMD("PING"); err != nil {
		t.Errorf("expected err=nil")
	}
	errs := startGoroutines(p, "PING")
	d.check("befor close", p, 1, 1)
	p.Close()

	timeout := time.After(2 * time.Second)
	for i := 0; i < cap(errs); i++ {
		select {
		case err := <-errs:
			t.Logf("%s", err)
		case <-timeout:
			t.Fatal("timeout")
		}
	}
	c.Close()
	d.check("done", p, 1, 0)
}
