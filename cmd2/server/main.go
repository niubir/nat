package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
}

var clients = make(map[string]*Client) // 存储客户端 ID 和连接
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	log.Println("Server started on :21200")
	log.Fatal(http.ListenAndServe(":21200", nil))
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	// 等待客户端发送其 ID
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("Error reading client ID:", err)
		return
	}

	clientID := string(msg)
	if _, exists := clients[clientID]; exists {
		log.Printf("Client ID %s already connected, closing connection.\n", clientID)
		return
	}

	clients[clientID] = &Client{ID: clientID, Conn: conn}
	log.Printf("Client %s connected\n", clientID)

	for {
		// 监听客户端发送的信令数据
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client %s disconnected\n", clientID)
			delete(clients, clientID)
			break
		}

		var signal map[string]interface{}
		err = json.Unmarshal(msg, &signal)
		if err != nil {
			log.Println("Invalid signal:", err)
			continue
		}

		targetID, ok := signal["target"].(string)
		if !ok {
			log.Println("Signal missing target ID")
			continue
		}

		targetClient, exists := clients[targetID]
		if !exists {
			log.Printf("Target client %s not found\n", targetID)
			continue
		}

		// 转发信令数据到目标客户端
		err = targetClient.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Printf("Failed to forward message to %s: %v\n", targetID, err)
		}
	}
}
