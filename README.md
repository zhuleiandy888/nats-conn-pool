# nats-conn-pool
A golang universal network connection pool for nats-server: [https://github.com/nats-io/nats-server](https://github.com/nats-io/nats-server).

## Basic Usage:

```go
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
		FactoryMaxRetry: 3,
		//连接最大空闲时间，超过该时间的连接 将会关闭，可避免空闲时连接EOF，自动失效的问题
		IdleTimeout: 600 * time.Second,
	}
	p, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}
	//从连接池中取得一个连接
	nc, err := p.Get()
	current := p.Len()
	fmt.Println("len=", current)
    //将连接放回连接池中
	p.Put(nc)
	//释放连接池中的所有连接
	p.Release()
	//查看当前连接中的数量
	current = p.Len()
	fmt.Println("len=", current)
	time.Sleep(200 * time.Second)
```

#### Remarks:
The connection pool implementation refers to pool [https://github.com/fatih/pool](https://github.com/fatih/pool) , [https://github.com/silenceper/pool](https://github.com/silenceper/pool), thanks.

## License

The MIT License (MIT) - see LICENSE for more details