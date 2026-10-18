package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PingHandler 健康检查与网络探活接口
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().Unix(),
	})
}
