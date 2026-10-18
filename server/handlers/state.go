package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"rm-judge-server/store"
)

// StateHandler 裁判系统状态机控制器
type StateHandler struct {
	store *store.MemoryStateStore
}

// NewStateHandler 创建控制器实例
func NewStateHandler(s *store.MemoryStateStore) *StateHandler {
	return &StateHandler{store: s}
}

// RegisterRoutes 注册裁判系统全部 API 路由
func (h *StateHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/ping", PingHandler)
		api.GET("/v1/ping", PingHandler)
		api.GET("/state", h.GetState)
		api.PUT("/state", h.PutState)
	}
}

// GetState 获取裁判系统比赛状态（附带 RFC 7232 ETag 标头）
func (h *StateHandler) GetState(c *gin.Context) {
	st, err := h.store.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// RFC 7232：向响应头注入强指纹 ETag
	c.Header("ETag", fmt.Sprintf("\"%d\"", st.Version))
	// 允许跨域请求读取 ETag 响应头
	c.Header("Access-Control-Expose-Headers", "ETag")

	// 直接返回保存的 JSON 文本内容，与前端 IndexedDB 保持字节级一致
	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(st.Data))
}

// PutState 保存裁判系统比赛状态（根据 If-Match 进行 OCC 乐观并发校验）
func (h *StateHandler) PutState(c *gin.Context) {
	// 1. 读取 RFC 7232 标准请求头 If-Match
	var expectedVersion int64
	ifMatch := c.GetHeader("If-Match")
	if ifMatch != "" {
		clean := strings.Trim(ifMatch, "\"")
		clean = strings.TrimPrefix(clean, "W/")
		clean = strings.Trim(clean, "\"")
		v, parseErr := strconv.ParseInt(clean, 10, 64)
		if parseErr == nil {
			expectedVersion = v
		}
	}

	// 2. 读取原始请求体
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body cannot be empty"})
		return
	}

	// 3. 提交至存储层执行原子 CAS 更新
	newState, err := h.store.Update(string(bodyBytes), expectedVersion)
	if err != nil {
		if errors.Is(err, store.ErrVersionConflict) {
			// 核心防线：命中 412 前置条件失败，拒绝对比赛数据进行踩踏覆盖！
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error":   "Precondition Failed: 裁判系统状态已被其他裁判更新，请先拉取最新比分！",
				"details": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. 更新成功，返回新版本 ETag 与数据
	c.Header("ETag", fmt.Sprintf("\"%d\"", newState.Version))
	c.Header("Access-Control-Expose-Headers", "ETag")
	c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(newState.Data))
}
