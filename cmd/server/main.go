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

type Peer struct {
	ID      string `json:"id" form:"id"`
	Address string `json:"address" form:"address"`
}

var cache = make(map[string]Peer) // 存储 peer 的 NAT 信息
var lock sync.Mutex

func main() {
	r := gin.Default()

	r.POST("/register", func(c *gin.Context) {
		var peer Peer
		if err := c.BindJSON(&peer); err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		lock.Lock()
		cache[peer.ID] = peer
		lock.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"message": "Registered successfully",
		})
	})

	r.GET("/get", func(c *gin.Context) {
		id := c.Query("id")

		lock.Lock()
		peer, exists := cache[id]
		lock.Unlock()

		if !exists {
			c.JSON(http.StatusNotFound, "peer not found")
			return
		}

		c.JSON(http.StatusOK, peer)
	})

	r.GET("/all", func(c *gin.Context) {
		var peers []Peer
		lock.Lock()
		for _, peer := range cache {
			peers = append(peers, peer)
		}
		lock.Unlock()

		c.JSON(http.StatusOK, peers)
	})

	r.Run(fmt.Sprintf(":%d", port))
}
