package main

import (
	"SimpleRedis/main/core/proto"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	IPPort := "127.0.0.1:9736"

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Hi Redis")
	tcpAddr, err := net.ResolveTCPAddr("tcp4", IPPort)
	checkError(err) // 通用的错误检查函数

	//建立连接 如果第二个参数(本地地址)为nil，会自动生成一个本地地址
	conn, err := net.DialTCP("tcp", nil, tcpAddr) // 建立与server的连接
	checkError(err)
	defer conn.Close() // 检查连接可用后,  此处定义进程退出时, 关闭连接

	for {
		fmt.Print(IPPort + "> ")
		text, _ := reader.ReadString('\n')
		//清除掉回车换行符
		text = strings.Replace(text, "\n", "", -1)
		send2Server(text, conn)

		buff := make([]byte, 1024)
		n, err := conn.Read(buff)
		resp, er := proto.DecodeFromBytes(buff)
		checkError(err)
		if n == 0 {
			fmt.Println(IPPort+"> ", "nil")
		} else if er == nil {
			fmt.Println(IPPort+">", string(resp.Value))
		} else {
			fmt.Println(IPPort+"> ", "error server response")
		}
	}

}
func send2Server(msg string, conn net.Conn) (n int, err error) {
	p, e := proto.EncodeCmd(msg)
	if e != nil {
		return 0, e
	}
	//fmt.Println("proto encode", p, string(p))
	n, err = conn.Write(p)
	return n, err
}
func checkError(err error) {
	if err != nil {
		log.Println("err ", err.Error())
		os.Exit(1)
	}
}
