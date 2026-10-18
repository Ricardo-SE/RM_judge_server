package models

import (
	"errors"
	"time"
)

// AppState 对应数据库中的 app_states 表记录
// 相比原始任务书的简单三字段，我们增加了 Version（版本号），为后续阶段二的乐观并发控制(OCC)打好地基
type AppState struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Data      string    `gorm:"type:text;not null" json:"data"`
	Version   int64     `gorm:"not null;default:1" json:"version"` // 乐观锁版本号
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// MatchStatus 比赛状态的枚举类型（用自定义类型增加编译期安全）
type MatchStatus string

const (
	MatchStatusReady     MatchStatus = "ready"     // 未开始/准备中
	MatchStatusOngoing   MatchStatus = "ongoing"   // 比赛进行中
	MatchStatusCompleted MatchStatus = "completed" // 比赛已完赛（终态）
)

// MatchState 裁判系统单场比赛的领域模型（带业务守恒校验）
type MatchState struct {
	MatchID   string      `json:"match_id"`
	RedScore  int         `json:"red_score"`
	BlueScore int         `json:"blue_score"`
	Status    MatchStatus `json:"status"`
}

var (
	// ErrNegativeScore 领域不变式报警：比分不能为负数
	ErrNegativeScore = errors.New("score cannot be negative: 比分不可为负数")
	// ErrIllegalRevert 状态机单向不可逆报警：完赛不可被倒灌撤回
	ErrIllegalRevert = errors.New("cannot revert completed match: 完赛记录不可倒灌修改")
)

// ValidateInvariants 校验领域核心不变式
func (m *MatchState) ValidateInvariants() error {
	if m.RedScore < 0 || m.BlueScore < 0 {
		return ErrNegativeScore
	}
	return nil
}

// CanTransitionTo 检查状态机转移是否合法（确保单向不可逆）
func (m *MatchState) CanTransitionTo(nextStatus MatchStatus) bool {
	// 如果当前已经是完赛状态，不允许退回为 ready 或 ongoing
	if m.Status == MatchStatusCompleted && nextStatus != MatchStatusCompleted {
		return false
	}
	return true
}
