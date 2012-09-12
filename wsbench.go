// This ipackage test the websocket endpoint that you specify
package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	"os"
	"time"
)

type Message struct {
	Action  string            `json:"action"`
	Payload map[string]string `json:"payload"`
}

// CreateWSConn Creates a new websocket connection
func CreateWsConn() (conn *websocket.Conn) {
	conn, err := websocket.Dial(Url, "", Origin)
	if err != nil {
		fmt.Println("Ooups!! I cannot create the websocket conn")
	}
	return
}

// CreateWsConnPool creates a connection pool of websocket the pool size
// is determined by the pool_size parameter.
func CreateWsConnPool(pool_size int) (ConnPool []*websocket.Conn) {
	fmt.Println("Create the connection pool")
	ConnPool = make([]*websocket.Conn, 0)
	for i := 0; i < pool_size; i++ {
		ConnPool = append(ConnPool, CreateWsConn())
	}
	return
}

// WsSender sends a Message as a JSON string on the websocket.
func WsSender(m *Message, conn *websocket.Conn) {
	if err := websocket.JSON.Send(conn, m); err != nil {
		fmt.Println("Ooups!! I can not Write msg into the websocket")
	}
}

// WsReader reads from a web socket and return pass the string to a channel
func WsReader(conn *websocket.Conn, c chan string) {
	var n int
	msg := make([]byte, 512)
	n, err := conn.Read(msg)
	if err != nil {
		panic(err)
	}
	c <- fmt.Sprintf("Receive : %s", msg[:n])
	return
}

// Subscribe to a channel
func Subscribe(channel string, wschan chan string, conn_pool []*websocket.Conn) {
	fmt.Println("Subscribe to channel:", channel)
	payloadDict := make(map[string]string)
	payloadDict["channel"] = channel
	msg := &Message{"subscribe", payloadDict}
	for i := range conn_pool {
		WsSender(msg, conn_pool[i])
		WsReader(conn_pool[i], wschan)
	}
}

// Publish a message m to the websocket channel wschan
func Publish(wschan, m string) {
	payloadDict := make(map[string]string)
	payloadDict["channel"] = wschan
	payloadDict["message"] = m
	msg := &Message{"publish", payloadDict}
	conn := CreateWsConn()
	WsSender(msg, conn)
}

// Command line handling
var (
	Origin       string
	Url          string
	ConnPoolSize int
	Help         bool
)

// Command line initialization
func init() {
	flag.StringVar(&Origin, "origin", "http://localhost/", "Origin http")
	flag.StringVar(&Url, "url", "ws://localhost:9000/pubsub/websocket", "websocket URL")
	flag.IntVar(&ConnPoolSize, "pool-size", 10, "Number of websocket")
	flag.BoolVar(&Help, "help", false, "Display this help message")
}

func usage() {
	fmt.Println("wsbench stress test your websocket server")
	fmt.Println("Usage: wsbench -pool-size=100 -url=ws://localhost:9000/pubsub/websocket")
	fmt.Println("Command line parameters")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if Help {
		flag.Usage()
	}
	//Create the connection pool and subscribe all the connections
	// the test channel
	ConnPool := CreateWsConnPool(ConnPoolSize)
	ws_rep_chan := make(chan string)
	go Subscribe("test", ws_rep_chan, ConnPool)

	timeout := time.After(20 * time.Second)
	for i := 0; i < len(ConnPool); i++ {
		select {
		case <-timeout:
			fmt.Println("Quit after timeout")
			return
		case rep := <-ws_rep_chan:
			fmt.Println(rep)
		}
	}
	time.Sleep(5 * time.Second)

	fmt.Println("publish 'hello' on the channel test")
	Publish("test", "hello")

	go func(conn_pool []*websocket.Conn, rep_chan chan string) {
		fmt.Println("Read the ws in the connection pool")
		for i := range ConnPool {
			WsReader(conn_pool[i], rep_chan)
		}
	}(ConnPool, ws_rep_chan)

	timeout = time.After(20 * time.Second)
	for i := 0; i < len(ConnPool); i++ {
		select {
		case <-timeout:
			fmt.Println("Quit after timeout")
			return
		case rep := <-ws_rep_chan:
			fmt.Println(rep)
		}
	}

}
