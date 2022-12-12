/*
 * @Notice: edit notice here
 * @Author: zhulei
 * @Date: 2022-12-02 20:57:58
 * @LastEditors: zhulei
 * @LastEditTime: 2022-12-12 14:26:31
 */
package pool

import (
	"errors"
	"fmt"
	"sync"
	"time"
	//"reflect"
)

var (

	//ErrFactoryMaxRetryReached 建立连接次数超限
	ErrFactoryMaxRetryReached = errors.New("FactoryMaxRetryReached")
)

// Config 连接池相关配置
type Config struct {
	//连接池中拥有的最小连接数
	InitialCap int
	//最大并发存活连接数
	MaxCap int
	// factory方法建立连接最大重试次数, 等于0不限制次数
	FactoryMaxRetry int
	//生成连接的方法
	Factory func() (interface{}, error)
	//关闭连接的方法
	Close func(interface{}) error
	//检查连接是否有效的方法
	Ping func(interface{}) bool
	//连接最大空闲时间，超过该事件则将失效
	IdleTimeout time.Duration
}

type idleConn struct {
	conn interface{}
	t    time.Time
}

// channelPool 存放连接信息
type channelPool struct {
	mu              sync.RWMutex
	conns           chan *idleConn
	factory         func() (interface{}, error)
	close           func(interface{}) error
	ping            func(interface{}) bool
	factoryMaxRetry int
	idleTimeout     time.Duration
	maxActive       int
}

// NewChannelPool 初始化连接
func NewChannelPool(poolConfig *Config) (Pool, error) {

	if poolConfig.InitialCap > poolConfig.MaxCap || poolConfig.InitialCap <= 0 || poolConfig.MaxCap <= 0 {
		return nil, errors.New("invalid capacity settings")
	}
	if poolConfig.Factory == nil {
		return nil, errors.New("invalid factory func settings")
	}
	if poolConfig.Close == nil {
		return nil, errors.New("invalid close func settings")
	}

	if poolConfig.FactoryMaxRetry < 0 || poolConfig.IdleTimeout <= 0 {
		return nil, errors.New("invalid FactoryMaxRetry or IdleTimeout argvs settings")
	}

	c := &channelPool{
		conns:           make(chan *idleConn, poolConfig.MaxCap),
		factory:         poolConfig.Factory,
		close:           poolConfig.Close,
		factoryMaxRetry: poolConfig.FactoryMaxRetry,
		idleTimeout:     poolConfig.IdleTimeout,
		maxActive:       poolConfig.MaxCap,
	}

	if poolConfig.Ping != nil {
		c.ping = poolConfig.Ping
	}

	for i := 0; i < poolConfig.InitialCap; i++ {
		conn, err := c.factory()
		if err != nil {
			c.Release()
			return nil, fmt.Errorf("factory() func failed to initialize the connection pool: %s", err)
		}
		c.conns <- &idleConn{conn: conn, t: time.Now()}
	}

	return c, nil
}

// getConns 获取所有连接
func (c *channelPool) getConns() chan *idleConn {
	c.mu.Lock()
	conns := c.conns
	c.mu.Unlock()
	return conns
}

// addConn 添加连接
// func (c *channelPool) addConn(conn interface{}) error {
// 	select {
// 	case c.conns <- &idleConn{conn: conn, t: time.Now()}: // Put conn in the channel unless it is full
// 	default:
// 		// fmt.Println("Channel full. Discarding value")
// 	}
// 	return nil
// }

// Get 从pool中取一个连接
func (c *channelPool) Get() (interface{}, error) {
	conns := c.getConns()
	if conns == nil {
		return nil, ErrClosed
	}
	retryCount := 0
	for {
		select {
		case wrapConn := <-conns:
			if wrapConn == nil {
				return nil, ErrClosed
			}
			//判断是否超时，超时则丢弃
			if timeout := c.idleTimeout; timeout > 0 {
				if wrapConn.t.Add(timeout).Before(time.Now()) {
					//丢弃并关闭该连接
					c.Close(wrapConn.conn)
					continue
				}
			}
			//判断是否失效，失效则丢弃，如果用户没有设定 ping 方法，就不检查
			if c.ping != nil {
				if !c.Ping(wrapConn.conn) {
					c.Close(wrapConn.conn)
					continue
				}
			}
			return wrapConn.conn, nil
		default:

			if c.factory == nil {
				return nil, ErrClosed
			}
			conn, err := c.factory()
			if err != nil {
				retryCount++
				if c.factoryMaxRetry != 0 && retryCount > c.factoryMaxRetry {
					return nil, ErrFactoryMaxRetryReached
				} else {
					continue
				}
			}
			return conn, nil
		}
	}
}

// Put 将连接放回pool中
func (c *channelPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	//判断是否失效，失效则丢弃，如果用户没有设定 ping 方法，就不检查
	if c.ping != nil {
		if !c.Ping(conn) {
			c.Close(conn)
			return errors.New("conn not active, close conn")
		}
	}

	c.mu.Lock()

	if c.conns == nil {
		c.mu.Unlock()
		return c.Close(conn)
	}

	select {
	case c.conns <- &idleConn{conn: conn, t: time.Now()}:
		c.mu.Unlock()
		return nil
	default:
		c.mu.Unlock()
		//连接池已满，直接关闭该连接
		return c.Close(conn)
	}

}

// Close 关闭单条连接
func (c *channelPool) Close(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.close == nil {
		return nil
	}
	return c.close(conn)
}

// Ping 检查单条连接是否有效
func (c *channelPool) Ping(conn interface{}) bool {
	if conn == nil {
		// return errors.New("connection is nil. rejecting")
		return false
	}
	return c.ping(conn)
}

// Release 释放连接池中所有连接
func (c *channelPool) Release() {
	c.mu.Lock()
	conns := c.conns
	c.conns = nil
	c.factory = nil
	c.ping = nil
	closeFun := c.close
	c.close = nil
	c.mu.Unlock()

	if conns == nil {
		return
	}

	close(conns)
	for wrapConn := range conns {
		//log.Printf("Type %v\n",reflect.TypeOf(wrapConn.conn))
		closeFun(wrapConn.conn)
	}
}

// Len 连接池中已有的连接
func (c *channelPool) Len() int {
	return len(c.getConns())
}
