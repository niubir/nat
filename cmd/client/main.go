package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	config *Config
)

type Config struct {
	Server    string `json:"server"`
	LocalPort int    `json:"localPort"`
	LocalID   string `json:"localID"`
}

func parseConfig() (*Config, error) {
	config, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	var data Config
	if err := json.Unmarshal(config, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func init() {
	var err error
	config, err = parseConfig()
	if err != nil {
		panic("parse config failed: " + err.Error())
	}
	fmt.Printf(
		"server: %s\nlocal port: %d\nlocal id: %s\n",
		config.Server, config.LocalPort, config.LocalID,
	)
}

func getExternalIP() (string, error) {
	resp, err := http.Get("https://www.ip.cn/api/index?ip=&type=0")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	ip := strings.TrimSpace(data.IP)
	if ip == "" {
		return "", fmt.Errorf("ip is empty")
	}
	return ip, nil
}

// 注册 NAT 信息到信令服务器
func register(id, address string) error {
	url := fmt.Sprintf("%s/register", config.Server)
	data := map[string]string{
		"id":      id,
		"address": address,
	}
	fmt.Println(data)
	payload, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to register")
	}
	return nil
}

// 获取对方的 NAT 信息
func getPeer(peerID string) (string, error) {
	url := fmt.Sprintf("%s/get?id=%s", config.Server, peerID)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	address := strings.TrimSpace(result["address"])
	if address == "" {
		return "", errors.New("peer not found")
	}
	return address, nil
}

// UDP NAT 穿透
func punchUDP(localAddr, peerAddr string) error {
	localUDPAddr, _ := net.ResolveUDPAddr("udp", localAddr)
	peerUDPAddr, _ := net.ResolveUDPAddr("udp", peerAddr)

	conn, err := net.DialUDP("udp", localUDPAddr, peerUDPAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 定时发送数据包以保持连接
	go func() {
		for {
			_, err := conn.Write([]byte("Hello from " + localAddr))
			if err != nil {
				fmt.Println("Failed to send message:", err)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// 接收数据
	buffer := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return err
		}
		fmt.Printf("Received from %s: %s\n", addr, string(buffer[:n]))
	}
}

func readInput(title string, must bool) string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(title)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}
	}
}

func main() {
	localAddr := fmt.Sprintf("0.0.0.0:%d", config.LocalPort)

	// 获取公网ip
	externalIP, err := getExternalIP()
	if err != nil {
		panic("get external ip failed: " + err.Error())
	} else {
		fmt.Println("get external ip success: ", externalIP)
	}
	externalAddr := fmt.Sprintf("%s:%d", externalIP, config.LocalPort)

	// 启动UDP监听
	udpAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		panic("resolve udp addr failed: " + err.Error())
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic("listen udp failed: " + err.Error())
	}
	defer conn.Close()

	// 注册到信令服务器
	if err := register(config.LocalID, externalAddr); err != nil {
		panic("register failed: " + err.Error())
	}

	connect := func() {
		// 输入对象ID
		peerID := readInput("input peer id:", true)
		// 获取对方的 NAT 信息
		peerExternalAddr, err := getPeer(peerID)
		if err != nil {
			fmt.Println("get peer failed: " + err.Error())
			return
		}
		// 开始 NAT 穿透
		punchUDP(localAddr, peerExternalAddr)
	}

	for {
		connect()
	}
}
