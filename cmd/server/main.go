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
		var param struct {
			ID      string `json:"id" form:"id"`
			Address string `json:"address" form:"address"`
		}
		if err := c.BindJSON(&param); err != nil {
			c.JSON(http.StatusNotFound, err.Error())
			return
		}

		lock.Lock()
		peers[param.ID] = param.Address
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
			c.JSON(http.StatusNotFound, "peer not found")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"address": address,
		})
	})

	r.GET("/all", func(c *gin.Context) {
		var ids []string

		lock.Lock()
		for id := range peers {
			ids = append(ids, id)
		}
		lock.Unlock()

		c.JSON(http.StatusOK, ids)
	})

	r.Run(fmt.Sprintf(":%d", port))
}
