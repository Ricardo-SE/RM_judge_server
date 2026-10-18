package models

import (
	"testing"
)

// TestMatchState_ValidateInvariants 验证比分非负的不变式铁律
func TestMatchState_ValidateInvariants(t *testing.T) {
	// 正常比赛数据
	validMatch := MatchState{
		MatchID:   "match_01",
		RedScore:  10,
		BlueScore: 5,
		Status:    MatchStatusOngoing,
	}
	if err := validMatch.ValidateInvariants(); err != nil {
		t.Fatalf("预期合法比分通过校验，但收到错误: %v", err)
	}

	// 异常数据：红队比分被改成了负数
	invalidMatch := MatchState{
		MatchID:   "match_02",
		RedScore:  -1,
		BlueScore: 5,
		Status:    MatchStatusOngoing,
	}
	if err := invalidMatch.ValidateInvariants(); err != ErrNegativeScore {
		t.Fatalf("预期拦截负数比分，实际返回: %v", err)
	}
}

// TestMatchState_CanTransitionTo 验证状态机单向不可逆（完赛不能回退）
func TestMatchState_CanTransitionTo(t *testing.T) {
	// 从准备中 -> 进行中：允许
	m := MatchState{Status: MatchStatusReady}
	if !m.CanTransitionTo(MatchStatusOngoing) {
		t.Errorf("状态转移 ready -> ongoing 应该被允许")
	}

	// 从进行中 -> 完赛：允许
	m.Status = MatchStatusOngoing
	if !m.CanTransitionTo(MatchStatusCompleted) {
		t.Errorf("状态转移 ongoing -> completed 应该被允许")
	}

	// 从完赛 -> 倒灌回退为进行中：必须拦截！
	m.Status = MatchStatusCompleted
	if m.CanTransitionTo(MatchStatusOngoing) {
		t.Errorf("状态转移 completed -> ongoing 必须被禁止！")
	}
}
