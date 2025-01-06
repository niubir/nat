package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	port = 21200
)

func init() {
	flag.IntVar(&port, "p", 21200, "Set server port, default use 21200")
	flag.Parse()
}

var peers = make(map[string]string) // 存储 peer 的 NAT 信息
var lock sync.Mutex

func main() {
	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		id := c.PostForm("id")
		address := c.PostForm("address")

		lock.Lock()
		peers[id] = address
		lock.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"message": "Registered successfully",
		})
	})

	r.GET("/get", func(c *gin.Context) {
		id := c.Query("id")

		lock.Lock()
		address, exists := peers[id]
		lock.Unlock()

		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Peer not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"address": address,
		})
	})

	r.Run(fmt.Sprintf(":%d", port))
}
