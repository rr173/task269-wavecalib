package model

import (
	"errors"
	"fmt"
)

// 领域错误集合：所有业务校验失败都映射为这些哨兵错误，
// 由 HTTP 层统一转换为 400/404/409 状态码。
var (
	// ErrNotFound 表示目标实体不存在。
	ErrNotFound = errors.New("实体不存在")
	// ErrConflict 表示违反唯一键或幂等约束。
	ErrConflict = errors.New("数据冲突")
	// ErrInvalidState 表示状态机流转非法。
	ErrInvalidState = errors.New("状态流转非法")
	// ErrInvalidInput 表示入参校验失败。
	ErrInvalidInput = errors.New("入参非法")
	// ErrSealed 表示对已封存运行执行修改操作。
	ErrSealed = errors.New("运行已封存，不可修改")
)

// ValidationError 携带具体校验失败原因的入参错误。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("字段 %s 校验失败：%s", e.Field, e.Message)
}

// Is 使 ValidationError 匹配 errors.Is(err, ErrInvalidInput)。
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// 状态机流转校验：返回 nil 表示允许 from -> to。
type TransitionRule func(from, to string) bool

// RunTransitions 定义观测运行允许的状态流转。
var RunTransitions = map[string]map[string]bool{
	RunStatusCollecting: {
		RunStatusAnalyzing: true,
		RunStatusSealed:    true,
	},
	RunStatusAnalyzing: {
		RunStatusReviewing: true,
		RunStatusSealed:    true,
	},
	RunStatusReviewing: {
		RunStatusConfirmed: true,
		RunStatusSealed:    true,
	},
	RunStatusConfirmed: {
		RunStatusSealed: true,
	},
	RunStatusSealed: {},
}

// CheckRunTransition 校验观测运行状态流转，非法时返回 ErrInvalidState。
func CheckRunTransition(from, to string) error {
	if from == to {
		return nil
	}
	allowed, ok := RunTransitions[from]
	if !ok || !allowed[to] {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidState, from, to)
	}
	return nil
}

// SnapshotTransitions 定义诊断快照状态流转。
var SnapshotTransitions = map[string]map[string]bool{
	SnapshotStatusDraft: {
		SnapshotStatusPub: true,
	},
	SnapshotStatusPub: {
		SnapshotStatusReplaced: true,
	},
	SnapshotStatusReplaced: {},
}

// CheckSnapshotTransition 校验快照状态流转。
func CheckSnapshotTransition(from, to string) error {
	if from == to {
		return nil
	}
	allowed, ok := SnapshotTransitions[from]
	if !ok || !allowed[to] {
		return fmt.Errorf("%w: 快照 %s -> %s", ErrInvalidState, from, to)
	}
	return nil
}
