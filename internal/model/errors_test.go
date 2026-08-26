package model

import (
	"errors"
	"testing"
)

func TestCheckRunTransition(t *testing.T) {
	valid := [][2]string{
		{RunStatusCollecting, RunStatusAnalyzing},
		{RunStatusCollecting, RunStatusSealed},
		{RunStatusAnalyzing, RunStatusReviewing},
		{RunStatusReviewing, RunStatusConfirmed},
		{RunStatusConfirmed, RunStatusSealed},
	}
	for _, tr := range valid {
		if err := CheckRunTransition(tr[0], tr[1]); err != nil {
			t.Fatalf("合法流转 %s -> %s 被拒绝: %v", tr[0], tr[1], err)
		}
	}
	invalid := [][2]string{
		{RunStatusSealed, RunStatusAnalyzing},  // 封存不可逆
		{RunStatusCollecting, RunStatusConfirmed}, // 跳级
		{RunStatusReviewing, RunStatusCollecting}, // 回退
	}
	for _, tr := range invalid {
		if err := CheckRunTransition(tr[0], tr[1]); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("非法流转 %s -> %s 未被拒绝: %v", tr[0], tr[1], err)
		}
	}
	// 同状态视为允许
	if err := CheckRunTransition(RunStatusSealed, RunStatusSealed); err != nil {
		t.Fatalf("同状态流转被拒绝: %v", err)
	}
}

func TestCheckSnapshotTransition(t *testing.T) {
	if err := CheckSnapshotTransition(SnapshotStatusDraft, SnapshotStatusPub); err != nil {
		t.Fatalf("草稿->发布 被拒绝: %v", err)
	}
	if err := CheckSnapshotTransition(SnapshotStatusPub, SnapshotStatusReplaced); err != nil {
		t.Fatalf("发布->替代 被拒绝: %v", err)
	}
	if err := CheckSnapshotTransition(SnapshotStatusPub, SnapshotStatusDraft); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("发布->草稿 未被拒绝: %v", err)
	}
}

func TestValidationErrorIs(t *testing.T) {
	err := &ValidationError{Field: "seq", Message: "必须为正整数"}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ValidationError 未匹配 ErrInvalidInput")
	}
}
