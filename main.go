package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"
)

var users map[uint64]*websocket.Conn
var index uint64

func WsHandler(c *gin.Context) {
	//token check

	upgrader := &websocket.Upgrader{
		// 解决跨域问题
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	fmt.Println("new conn ", c.Request.RemoteAddr)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "websocket upgrade error")
		return
	}
	if nil == conn {
		c.String(http.StatusInternalServerError, "websocket upgrade error")
		return
	}
	atomic.AddUint64(&index, 1)
	users[index] = conn

	defer func() {
		if v := recover(); v != nil {
			log.Default().Printf("writePump Panic!!!! %s \n", string(debug.Stack()))
		}
	}()

	defer func() {
		if v := recover(); v != nil {
			// ChatLog.Errorf("readPump Panic!!!! panic:%+v info:%s", v, string(debug.Stack()))
			log.Default().Printf("readPump Panic!!!!,%s \n", string(debug.Stack()))
		}

	}()

	conn.SetReadDeadline(time.Time{}) //默认无超时.阻塞等待
	func() {
		for {
			t, message, err := conn.ReadMessage()
			if err != nil {
				//break loop
				fmt.Println(err.Error())
				return
			}
			prefix := fmt.Sprintf("[%s]:", conn.RemoteAddr().String())
			msg := append([]byte(prefix), message[:]...)
			for _, user := range users {
				user.WriteMessage(t, msg)
			}
		}
	}()

	conn.Close()

	return
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.StaticFS("/static", http.Dir("./static"))
	router.GET("/", func(c *gin.Context) {
		files, err := template.ParseFiles("./static/html/chat.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		files.Execute(c.Writer, nil)
	})
	router.GET("/ws", WsHandler)
	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           router,
		ReadHeaderTimeout: 2 * time.Second,
	}
	fmt.Println("server start  0.0.0.0:8080")
	users = make(map[uint64]*websocket.Conn)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}
