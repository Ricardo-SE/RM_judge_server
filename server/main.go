package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"rm-judge-server/handlers"
	"rm-judge-server/store"
)

func main() {
	r := gin.Default()

	// 配置跨域中间件，放行所有前端来源，并支持 If-Match 请求头与 ETag 暴露
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "If-Match"}
	corsConfig.ExposeHeaders = []string{"ETag"}
	r.Use(cors.New(corsConfig))

	// 初始化内存并发安全存储底座
	s := store.NewMemoryStateStore()
	h := handlers.NewStateHandler(s)
	h.RegisterRoutes(r)

	log.Println("[miniRM Server] 裁判系统后端服务正在启动，监听端口 :8080 ...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
