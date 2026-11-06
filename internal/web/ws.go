package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// ------------- WebSocket 设置 -------------
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var wsClients = make(map[*websocket.Conn]bool)
var msgChan = make(chan wsMsg, 100)

func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS Upgrade:", err)
		return
	}
	wsClients[conn] = true
	log.Println("客户端已连接日志 WebSocket")

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			delete(wsClients, conn)
			conn.Close()
			break
		}
	}
}

type wsMsg struct {
	Cmd string
	Msg string
}

// 提交网页日志
func WsSend(cmd, msg string) {
	msgChan <- wsMsg{cmd, msg}
}

// 后端广播日志给所有网页上的日志框
func BroadCaster() {
	for msg := range msgChan {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("BroadCaster panic:", r)
				}
			}()
			data, err := json.Marshal(msg)
			if err != nil {
				return
			}
			for c := range wsClients {
				if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Println("WS write error:", err)
					delete(wsClients, c)
					c.Close()
				}
			}
		}()
	}
}
