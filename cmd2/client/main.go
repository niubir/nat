package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

func main() {
	clientID := flag.String("id", "", "Unique client ID")
	serverAddr := flag.String("server", "localhost:8080", "Signaling server address")
	flag.Parse()

	if *clientID == "" {
		log.Fatal("Client ID is required")
	}
	fmt.Println(*clientID, *serverAddr)

	// 连接信令服务器
	u := url.URL{Scheme: "ws", Host: *serverAddr, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Failed to connect to signaling server:", err)
	}
	defer conn.Close()

	// 发送客户端 ID 到服务器
	err = conn.WriteMessage(websocket.TextMessage, []byte(*clientID))
	if err != nil {
		log.Fatal("Failed to send client ID:", err)
	}

	// 创建 WebRTC PeerConnection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatal(err)
	}
	defer peerConnection.Close()

	// 创建数据通道
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Fatal(err)
	}

	dataChannel.OnOpen(func() {
		fmt.Println("Data channel opened!")
		go func() {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				dataChannel.SendText(scanner.Text())
			}
		}()
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Received message: %s\n", string(msg.Data))
	})

	// 监听 ICE 候选
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			candidateJSON := candidate.ToJSON()
			signal := map[string]interface{}{
				"type":      "candidate",
				"candidate": candidateJSON,
				"target":    "", // 目标 ID
			}
			message, _ := json.Marshal(signal)
			conn.WriteMessage(websocket.TextMessage, message)
		}
	})

	// 接收信令数据
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Connection closed:", err)
				return
			}

			var signal map[string]interface{}
			err = json.Unmarshal(msg, &signal)
			if err != nil {
				log.Println("Invalid signal:", err)
				continue
			}

			switch signal["type"] {
			case "offer":
				var offer webrtc.SessionDescription
				offer.Type = webrtc.SDPTypeOffer
				offer.SDP = signal["sdp"].(string)
				peerConnection.SetRemoteDescription(offer)

				answer, _ := peerConnection.CreateAnswer(nil)
				peerConnection.SetLocalDescription(answer)

				signal := map[string]interface{}{
					"type":   "answer",
					"sdp":    answer.SDP,
					"target": signal["source"],
				}
				message, _ := json.Marshal(signal)
				conn.WriteMessage(websocket.TextMessage, message)

			case "answer":
				var answer webrtc.SessionDescription
				answer.Type = webrtc.SDPTypeAnswer
				answer.SDP = signal["sdp"].(string)
				peerConnection.SetRemoteDescription(answer)

			case "candidate":
				var candidate webrtc.ICECandidateInit
				candidate.Candidate = signal["candidate"].(map[string]interface{})["candidate"].(string)
				peerConnection.AddICECandidate(candidate)
			}
		}
	}()

	// 创建 SDP Offer 并指定目标
	var targetID string
	fmt.Print("Enter target ID: ")
	fmt.Scanln(&targetID)

	offer, _ := peerConnection.CreateOffer(nil)
	peerConnection.SetLocalDescription(offer)

	signal := map[string]interface{}{
		"type":   "offer",
		"sdp":    offer.SDP,
		"target": targetID,
	}
	message, _ := json.Marshal(signal)
	conn.WriteMessage(websocket.TextMessage, message)

	select {}
}
