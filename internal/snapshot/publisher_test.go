package snapshot

import (
	"context"
	"testing"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// TestPublish_ReplacesPriorPublishedSnapshot 验证：同一运行连续发布两个快照版本后，
// 旧版本应被置为"替代"，仅保留一个"发布"态快照（版本锁定生效）。
func TestPublish_ReplacesPriorPublishedSnapshot(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("打开内存库: %v", err)
	}
	defer s.Close()
	p := NewPublisher(s)

	const runID = "run-repl"
	if err := s.CreateRun(ctx, &model.ObservationRun{ID: runID, Name: "r", TelescopeID: "tel", Status: model.RunStatusReviewing}); err != nil {
		t.Fatalf("建运行: %v", err)
	}

	mkSnap := func() *model.DiagnosisSnapshot {
		snap, err := p.Create(ctx, runID, "mat", Payload{RunName: "r"})
		if err != nil {
			t.Fatalf("创建快照: %v", err)
		}
		return snap
	}

	// 发布 v1
	v1 := mkSnap()
	if _, err := p.Publish(ctx, v1.ID); err != nil {
		t.Fatalf("发布 v1: %v", err)
	}
	// 发布 v2，应替代 v1
	v2 := mkSnap()
	if _, err := p.Publish(ctx, v2.ID); err != nil {
		t.Fatalf("发布 v2: %v", err)
	}

	list, err := p.List(ctx, runID)
	if err != nil {
		t.Fatalf("列出快照: %v", err)
	}
	var pub, replaced int
	for _, s := range list {
		switch s.Status {
		case model.SnapshotStatusPub:
			pub++
		case model.SnapshotStatusReplaced:
			replaced++
		}
	}
	if pub != 1 {
		t.Fatalf("已发布快照应为 1 个，实际 %d（版本锁定未生效）", pub)
	}
	if replaced != 1 {
		t.Fatalf("被替代快照应为 1 个，实际 %d", replaced)
	}

	// 回查 v1，应为"替代"
	got, err := p.Get(ctx, v1.ID)
	if err != nil {
		t.Fatalf("回读 v1: %v", err)
	}
	if got.Status != model.SnapshotStatusReplaced {
		t.Fatalf("v1 应为 替代，实际 %s", got.Status)
	}
}
