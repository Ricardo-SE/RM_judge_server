package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"rm-judge-server/models"
)

var (
	// ErrStateNotFound 状态不存在
	ErrStateNotFound = errors.New("state not found: 比赛状态不存在")
	// ErrVersionConflict OCC 乐观并发冲突
	ErrVersionConflict = errors.New("version conflict: 比赛状态已被其他裁判更新，提交版本已过期")
)

// MemoryStateStore 裁判系统内存并发安全状态引擎
type MemoryStateStore struct {
	mu    sync.RWMutex
	state *models.AppState
}

// NewMemoryStateStore 初始化默认比赛状态（初始版本为 1）
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		state: &models.AppState{
			ID:        1,
			Data:      "{}",
			Version:   1,
			UpdatedAt: time.Now(),
		},
	}
}

// Get 读取当前比赛状态（返回独立副本）
func (s *MemoryStateStore) Get() (*models.AppState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyState := *s.state
	return &copyState, nil
}

// Update 带有 OCC 乐观锁的更新操作
// 只有当客户端提交的 expectedVersion 与当前内存中的 Version 一致时才允许写入并自增版本
func (s *MemoryStateStore) Update(newData string, expectedVersion int64) (*models.AppState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// RFC 7232 / OCC 前置条件判定：
	// 若客户端指定了期望版本号，且当前版本与期望不符，当场拒绝！
	if expectedVersion > 0 && s.state.Version != expectedVersion {
		return nil, fmt.Errorf("%w: 当前服务端版本为 %d, 客户端提交版本为 %d", ErrVersionConflict, s.state.Version, expectedVersion)
	}

	s.state.Data = newData
	s.state.Version++
	s.state.UpdatedAt = time.Now()

	copyState := *s.state
	return &copyState, nil
}
