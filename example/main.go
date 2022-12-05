/*
 * @Notice: edit notice here
 * @Author: zhulei
 * @Date: 2022-12-02 20:33:37
 * @LastEditors: zhulei
 * @LastEditTime: 2022-12-05 15:28:23
 */
package main

import (
	// "fmt"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	pool "github.com/zhuleiandy888/nats-conn-pool"
)

const (
	NATS_SERVER string = "nats://192.168.1.1:4222"
	// 连接nats-server 超时时间
	DEFAULT_CONNECT_TIMEOUT int = 1
	// MAX_RECONNECTS          int    = 3
	// RECONNECT_WAIT          int    = 3
	PING_INTERVAL         int    = 20
	MAX_PINGS_OUTSTANDING int    = 3
	TOKEN                 string = "xxxxxxxxxx"
)

func RandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func client() {

	//factory 创建连接的方法
	factory := func() (interface{}, error) {
		return nats.Connect(NATS_SERVER, nats.Token(TOKEN),
			nats.Timeout(time.Duration(DEFAULT_CONNECT_TIMEOUT)*time.Second),
			// nats.RetryOnFailedConnect(true),
			// nats.MaxReconnects(MAX_RECONNECTS),
			// nats.ReconnectWait(time.Duration(RECONNECT_WAIT)*time.Second),
			// nats.ReconnectJitter(time.Second*3, time.Second*3),
			//到达 MAX_PINGS_OUTSTANDING 次间隔时间（PING_INTERVAL * MAX_PINGS_OUTSTANDING 秒）仍然没有收到回复pong的关闭这个连接
			nats.PingInterval(time.Duration(PING_INTERVAL)*time.Second),
			nats.MaxPingsOutstanding(MAX_PINGS_OUTSTANDING),
			// nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			// 	// handle disconnect error event

			// }),
			// nats.ReconnectHandler(func(nc *nats.Conn) {
			// 	// handle reconnect event

			// }),
			// nats.ClosedHandler(func(nc *nats.Conn) {
			// 	// handle close event

			// }),
		)
	}

	//close 关闭连接的方法
	close := func(v interface{}) error { return v.(*nats.Conn).Drain() }

	ping := func(v interface{}) bool { return v.(*nats.Conn).IsConnected() }

	//创建一个连接池： 初始化100，最大连接1000
	poolConfig := &pool.Config{
		InitialCap: 100,
		MaxCap:     1000,
		Factory:    factory,
		Close:      close,
		Ping:       ping,
		//连接最大空闲时间，超过该时间的连接 将会关闭，可避免空闲时连接EOF，自动失效的问题
		IdleTimeout: 600 * time.Second,
	}
	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}

	//从连接池中取得一个连接
	// nc, err := p.Get()

	current := p.Len()
	fmt.Println("len=", current)
	// time.Sleep(6 * time.Second)
	//do something
	//conn=v.(net.Conn)
	count := 0

	for {
		count++
		if count > 10000 {
			break
		}
		if count%1000 == 0 {
			time.Sleep(1 * time.Second)
		}
		go pubMsg(p, "subject_1", []byte("Hello World--"+strconv.Itoa(count)+"--"+RandStr(8)))
		go pubMsg(p, "subject_2", []byte("Hello World--"+strconv.Itoa(count)+"--"+RandStr(8)))
		go pubMsg(p, "subject_3", []byte("Hello World--"+strconv.Itoa(count)+"--"+RandStr(8)))

	}

	//将连接放回连接池中
	// p.Put(nc)

	//释放连接池中的所有连接
	//p.Release()

	//查看当前连接中的数量
	// current = p.Len()
	// fmt.Println("len=", current)
	// time.Sleep(10 * time.Second)
}

func pubMsg(pl pool.Pool, sub string, msg []byte) {
	nc, err := pl.Get()
	if err != nil {
		fmt.Println("err0=", err)
	}
	// current := pl.Len()
	// fmt.Println("len=", current)
	err = nc.(*nats.Conn).Publish(sub, msg)
	if err != nil {
		fmt.Println("err1=", err)
	}
	err = nc.(*nats.Conn).Flush()
	if err != nil {
		fmt.Println("err2=", err)
	}

	pl.Put(nc)
	current := pl.Len()
	fmt.Println("len=", current)
}

func main() {

	client()

	time.Sleep(120 * time.Second)

}
