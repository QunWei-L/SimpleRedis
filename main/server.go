package main

import (
	"SimpleRedis/main/core"
	"github.com/tidwall/evio"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	DefaultAofFile = "./redis.aof"
)

// 服务端实例
var redisServer = new(core.Server)

func main() {
	/*
	* 服务器初始化
	* 1. 初始化变量 (默认值)
	* 2. 加载配置选项 (包含用户自定义)
	* 3. 初始化数据结构 (根据最新的配置值, 初始化数据结构)
	* 4. 还原数据库状态 (持久化数据)
	* 5. 执行事件循环(event-loop), 可以接受客户端的连接请求, 并处理客户端发来的 命令请求
	 */

	argv := os.Args
	argc := len(argv)
	if argc >= 2 {
		// do something, some options
	}

	// 初始化服务端实例
	initServer()

	// 监听信号 平滑退出. 例如退出时, 进行持久化AOF
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go sigHandler(c)

	//自定义事件处理
	handleEvents()
	//监听端口
	if err := evio.Serve(redisServer.Events, "tcp://127.0.0.1:9736"); err != nil {
		panic(err.Error())
	}
}

func handleEvents() {
	redisServer.Events.Serving = func(srv evio.Server) (action evio.Action) {
		log.Printf("redis server started on port %d (loops: %d)", 5000, srv.NumLoops)
		return
	}
	redisServer.Events.Opened = func(ec evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
		log.Printf("opened: %v\n", ec.RemoteAddr())
		//ec.SetContext(&conn{})
		return
	}
	redisServer.Events.Closed = func(ec evio.Conn, err error) (action evio.Action) {
		log.Printf("closed: %v\n", ec.RemoteAddr())
		return
	}

	redisServer.Events.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
		clientContext := c.Context()
		if clientContext == nil {
			clientContext = redisServer.CreateClient()
			c.SetContext(clientContext)
		}
		client, ok := clientContext.(*core.Client)
		if !ok {

		}
		err := client.ReadQueryFromInput(in) // 从连接中, 读取数据, 写到client的QueryBuf字段上

		if err != nil {
			log.Println("ReadQueryFromInput error: ", err)
			return
		}
		err = client.ProcessInputBuffer() // 命令格式化, 写入client的Argv数组
		if err != nil {
			log.Println("ProcessInputBuffer err", err)
			return
		}
		redisServer.ProcessCommand(client) // 执行命令,  将结果写入client.Buf字段
		out = responseOutput(client)       // 执行结果返回 请求方
		return
	}
}

// 请求处理方法
func handle(conn net.Conn) {
	c := redisServer.CreateClient() // (请求到来) 创建一个client结构体, 初始化默认值, 用来保存当前连接
	for {
		err := c.ReadQueryFromClient(conn) // 从连接中, 读取数据, 写到client的QueryBuf字段上

		if err != nil {
			log.Println("readQueryFromClient error: ", err)
			return
		}
		err = c.ProcessInputBuffer() // 命令格式化, 写入client的Argv数组
		if err != nil {
			log.Println("ProcessInputBuffer err", err)
			return
		}
		redisServer.ProcessCommand(c) // 执行命令,  将结果写入client.Buf字段
		responseConn(conn, c)         // 执行结果返回 请求方
	}
}

// 响应返回给客户端
func responseConn(conn net.Conn, c *core.Client) {
	conn.Write([]byte(c.Buf))
}

// 获取请求回复
func responseOutput(c *core.Client) []byte {
	return []byte(c.Buf)
}

// 初始化服务端实例
func initServer() {
	redisServer.Pid = os.Getpid()
	redisServer.DbNum = 16
	initDb() // 根据DbNum参数, 进行相应db初始化
	redisServer.Start = time.Now().UnixNano() / 1000000
	//var getf server.CmdFun
	redisServer.AofFilename = DefaultAofFile

	getCommand := &core.RedisCommand{Name: "get", Proc: core.GetCommand}
	setCommand := &core.RedisCommand{Name: "set", Proc: core.SetCommand}

	// 这是一个kv映射表, ->  "命令表", 用来记录: 命令的名字 / 对应的函数指针 / 参数校验
	redisServer.Commands = map[string]*core.RedisCommand{
		"get": getCommand,
		"set": setCommand,
	}
	redisServer.Events.NumLoops = 1 //单进程模式
	LoadData()
}

// 初始化db
func initDb() {
	redisServer.Db = make([]*core.RedisDb, redisServer.DbNum)
	for i := 0; i < redisServer.DbNum; i++ {
		redisServer.Db[i] = new(core.RedisDb)
		// 初始化哈希表, 申请一定的空间
		redisServer.Db[i].Dict = make(map[string]*core.RedisObject, 100)
	}
	log.Println("init db begin-->", redisServer.Db)
}
func LoadData() {
	c := redisServer.CreateClient()
	c.FakeFlag = true
	pros := core.ReadAof(redisServer.AofFilename)
	for _, v := range pros {
		c.QueryBuf = string(v)
		err := c.ProcessInputBuffer()
		if err != nil {
			log.Println("ProcessInputBuffer err", err)
		}
		redisServer.ProcessCommand(c)
	}
}

// 监听退出信号, 进行处理
func sigHandler(c chan os.Signal) {
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			exitHandler()
		default:
			log.Println("signal ", s)
		}
	}
}

// 平滑退出
func exitHandler() {
	log.Println("exiting smoothly ...")
	log.Println("bye ")
	os.Exit(0)
}
