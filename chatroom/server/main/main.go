package main
import (
	"fmt"
	"net"
	"time"
	"go_code/chatroom/server/model"
)

//处理和客户端的通讯
func process(conn net.Conn) {
	//这里需要延时关闭conn
	defer conn.Close()
	
	//这里调用总控, 创建一个
	processor := &Processor{
		Conn : conn,
	}
	err := processor.process2()
	if err != nil {
		fmt.Println("客户端和服务器通讯协程错误=err", err)
		return 
	}
}

func init() {
	//当服务器启动时，我们就去初始化我们的redis的连接池
	initPool("localhost:6379", 16, 0, 300 * time.Second)
	initUserDao()
}

//这里我们编写一个函数，完成对UserDao的初始化任务
func initUserDao() {
	//这里的pool 本身就是一个全局的变量
	//这里需要注意一个初始化顺序问题
	//initPool, 在 initUserDao
	model.MyUserDao = model.NewUserDao(pool)
}

func main() {
	
	//提示信息
	fmt.Println("服务器[新的结构]在8889端口监听....")
	listen, err := net.Listen("tcp", "0.0.0.0:8889")
	defer listen.Close()
	if err != nil {
		fmt.Println("net.Listen err=", err)
		return 
	}
	//一旦监听成功，就等待客户端来链接服务器
	for {
		fmt.Println("等待客户端来链接服务器.....")
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("listen.Accept err=" ,err)
		}

		//一旦链接成功，则启动一个协程和客户端保持通讯。。
		go process(conn)
	}
}