# mysql prepare & golang
>背景：最开始听说prepare是说这个能防sql注入，但是对其原理并不清楚，最近周末有一个业务使用了prepare，导致连接数据库超时较多，业务有损，因此记录下，防止以后再次踩坑。

## 1 mysql prepare原理&优势劣势分析
>mysql执行sql过程：词法分析->语法分析->语义分析->执行计划优化->执行

>其中 词法分析->语法分析 称之为硬解析。对于只是参数不同，其他均相同的sql，这些sql总的执行时间不同，但是硬解析的时间是相同的。

>所以，如果可以将硬解析的这块抽象出来，这样就能节省一些时间了。

>prepare的出现就是为了优化硬解析的问题，Prepare服务端的执行过程如下

> 1 prepare接收客户端发过来的"?"的sql，硬解析得到语法树，缓存在线程所在的prepare statement cache中，此cache是一个hash map，key是stmt->id，然后将stmt->id返回给客户端

> 2 execute接收客户端发过来的stmt->id和参数等信息。注意，此时客户端无需再次发送sql过来，只需要发送stmt->id即可。服务器知道stmt->id后，从prepare statement cache中得到硬解析后的结果，将参数设置一下，即可执行后续了。

>prepare优势：

> execute阶段中，可以节省掉硬解析的时间。如果一类sql，很多次执行，那么就能节省出来很多时间。所以，prepare适用于执行频繁的SQL。

> 防sql注入方面。由于sql语句和参数值是分开发送的，拼接是由服务器拼接，且只拼参数，所以攻击者通过在数据中混合sql是无法生效的。

>prepare缺点：

> 不频繁的sql多一次网络消耗。如果sql只执行一次，且以prepare的方式执行，那么sql执行需两次与服务器交互（Prepare和execute）, 而以普通（非prepare）方式，只需要一次交互。

> 但如果下游有一层DB proxy的话，可能效果会变差。原因是，proxy连接池和真正mysql连接池是不同的，如果prepare的query proxy不放入连接池，每次请求都是新建立连接、然后再次回放请求set names 、use db指令，最后发sql执行。这样效果会变差很多。我们遇到的就是这种情况。

## 2 golang中使用?号的是prepare么？InterpolateParams参数的作用是什么？
>golang测试代码
>先说结论，不一定。
>当InterpolateParams参数如果设置为true，允许驱动进行对sql进行插值，说明不使用prepared statement，且防sql注入。
>当InterpolateParams参数如果设置为false，不允许驱动进行对sql进行插值。

```golang
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)
func getConn() (*sql.DB, error) {
    dsn := ...
	conn, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}
	
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(60)

	return conn, nil
}

func main() {
	c, err :=  getConn()
	if err != nil {
		fmt.Println(err)
		return
    }
    // prepare
	row := c.QueryRow("select id from mytable where id = ?", 80000000001)
    // no prepare
	///row := c.QueryRow("select id from mytable where id = 80000000001")
	fmt.Println(row)
}
```

>核心函数queryDC，关键是在下面函数

```golang
// queryDC executes a query on the given connection.
// The connection gets released by the releaseConn function.
// The ctx context is from a query method and the txctx context is from an
// optional transaction context.
func (db *DB) queryDC(ctx, txctx context.Context, dc *driverConn, releaseConn func(error), query string, args []interface{}) (*Rows, error) {
	queryerCtx, ok := dc.ci.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = dc.ci.(driver.Queryer)
	}
	if ok {
		var nvdargs []driver.NamedValue
		var rowsi driver.Rows
		var err error
		withLock(dc, func() {
			nvdargs, err = driverArgsConnLocked(dc.ci, nil, args)
			if err != nil {
				return
			}
			rowsi, err = ctxDriverQuery(ctx, queryerCtx, queryer, query, nvdargs)
        })
        // 如果err返回是driver.ErrSkip，那么走底下的prepare逻辑
        // 如果err不是driver.ErrSkip，走 if err != driver.ErrSkip 逻辑， 直接返回
		if err != driver.ErrSkip {
			if err != nil {
				releaseConn(err)
				return nil, err
			}
			// Note: ownership of dc passes to the *Rows, to be freed
			// with releaseConn.
			rows := &Rows{
				dc:          dc,
				releaseConn: releaseConn,
				rowsi:       rowsi,
			}
			rows.initContextClose(ctx, txctx)
			return rows, nil
		}
	}

    // prepare逻辑
	var si driver.Stmt
	var err error
	withLock(dc, func() {
		si, err = ctxDriverPrepare(ctx, dc.ci, query)
	})
	if err != nil {
		releaseConn(err)
		return nil, err
	}

	ds := &driverStmt{Locker: dc, si: si}
	rowsi, err := rowsiFromStatement(ctx, dc.ci, ds, args...)
	if err != nil {
		ds.Close()
		releaseConn(err)
		return nil, err
	}

	// Note: ownership of ci passes to the *Rows, to be freed
	// with releaseConn.
	rows := &Rows{
		dc:          dc,
		releaseConn: releaseConn,
		rowsi:       rowsi,
		closeStmt:   ds,
	}
	rows.initContextClose(ctx, txctx)
	return rows, nil
}

```

>核心函数ctxDriverQuery

```golang
func ctxDriverQuery(ctx context.Context, queryerCtx driver.QueryerContext, queryer driver.Queryer, query string, nvdargs []driver.NamedValue) (driver.Rows, error) {
	if queryerCtx != nil {
        // 这个逻辑
		return queryerCtx.QueryContext(ctx, query, nvdargs)
	}
	dargs, err := namedValueToValue(nvdargs)
	if err != nil {
		return nil, err
	}
```

```golang
//github.com/go-sql-driver/mysql@v1.4.1/connection_go18.go
func (mc *mysqlConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}

	if err := mc.watchCancel(ctx); err != nil {
		return nil, err
	}

    // 走到这个逻辑，进入
	rows, err := mc.query(query, dargs)
	if err != nil {
		mc.finish()
		return nil, err
	}
	rows.finish = mc.finish
	return rows, err
}

// github.com/go-sql-driver/mysql@v1.4.1/connection.go
func (mc *mysqlConn) query(query string, args []driver.Value) (*textRows, error) {
	if mc.closed.IsSet() {
		errLog.Print(ErrInvalidConn)
		return nil, driver.ErrBadConn
    }
    // 参数不为空
	if len(args) != 0 {
        // 如果InterpolateParams不设置，那么返回driver.ErrSkip，走prepare逻辑
		if !mc.cfg.InterpolateParams {
			return nil, driver.ErrSkip
		}
        // try client-side prepare to reduce roundtrip
        // 转义参数，里面包含特殊字符处理，
		prepared, err := mc.interpolateParams(query, args)
		if err != nil {
			return nil, err
		}
		query = prepared
	}
	// Send command
	err := mc.writeCommandPacketStr(comQuery, query)
```

```golang
func (mc *mysqlConn) interpolateParams(query string, args []driver.Value) (string, error) {
	// Number of ? should be same to len(args)
	if strings.Count(query, "?") != len(args) {
		return "", driver.ErrSkip
	}

	buf := mc.buf.takeCompleteBuffer()
	if buf == nil {
		// can not take the buffer. Something must be wrong with the connection
		errLog.Print(ErrBusyBuffer)
		return "", ErrInvalidConn
	}
	buf = buf[:0]
	argPos := 0

	for i := 0; i < len(query); i++ {
		q := strings.IndexByte(query[i:], '?')
		if q == -1 {
			buf = append(buf, query[i:]...)
			break
		}
		buf = append(buf, query[i:i+q]...)
		i += q

		arg := args[argPos]
		argPos++

		if arg == nil {
			buf = append(buf, "NULL"...)
			continue
		}

		switch v := arg.(type) {
		case int64:
			buf = strconv.AppendInt(buf, v, 10)
		case float64:
			buf = strconv.AppendFloat(buf, v, 'g', -1, 64)
		case bool:
			if v {
				buf = append(buf, '1')
			} else {
				buf = append(buf, '0')
			}
		case time.Time:
			if v.IsZero() {
				buf = append(buf, "'0000-00-00'"...)
			} else {
				v := v.In(mc.cfg.Loc)
				v = v.Add(time.Nanosecond * 500) // To round under microsecond
				year := v.Year()
				year100 := year / 100
				year1 := year % 100
				month := v.Month()
				day := v.Day()
				hour := v.Hour()
				minute := v.Minute()
				second := v.Second()
				micro := v.Nanosecond() / 1000

				buf = append(buf, []byte{
					'\'',
					digits10[year100], digits01[year100],
					digits10[year1], digits01[year1],
					'-',
					digits10[month], digits01[month],
					'-',
					digits10[day], digits01[day],
					' ',
					digits10[hour], digits01[hour],
					':',
					digits10[minute], digits01[minute],
					':',
					digits10[second], digits01[second],
				}...)

				if micro != 0 {
					micro10000 := micro / 10000
					micro100 := micro / 100 % 100
					micro1 := micro % 100
					buf = append(buf, []byte{
						'.',
						digits10[micro10000], digits01[micro10000],
						digits10[micro100], digits01[micro100],
						digits10[micro1], digits01[micro1],
					}...)
				}
				buf = append(buf, '\'')
			}
		case []byte:
			if v == nil {
				buf = append(buf, "NULL"...)
			} else {
				buf = append(buf, "_binary'"...)
				if mc.status&statusNoBackslashEscapes == 0 {
					buf = escapeBytesBackslash(buf, v)
				} else {
					buf = escapeBytesQuotes(buf, v)
				}
				buf = append(buf, '\'')
			}
        case string:
        // 特殊字符转义，防止sql注入
			buf = append(buf, '\'')
			if mc.status&statusNoBackslashEscapes == 0 {
				buf = escapeStringBackslash(buf, v)
			} else {
				buf = escapeStringQuotes(buf, v)
			}
			buf = append(buf, '\'')
		default:
			return "", driver.ErrSkip
		}

		if len(buf)+4 > mc.maxAllowedPacket {
			return "", driver.ErrSkip
		}
	}
	if argPos != len(args) {
		return "", driver.ErrSkip
	}
	return string(buf), nil
}
```

## 参考资料
```
database/sql 一点深入理解 https://michaelyou.github.io/2018/03/30/database-sql-%E4%B8%80%E7%82%B9%E6%B7%B1%E5%85%A5%E7%90%86%E8%A7%A3/
Golang Mysql笔记（三）--- Prepared剖析 https://www.jianshu.com/p/ee0d2e7bef54
MySQL通信协议 https://jin-yang.github.io/post/mysql-protocol.html
```