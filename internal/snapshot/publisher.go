// Package snapshot 负责诊断快照的创建、发布与替代。
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// Publisher 是诊断快照发布器。
type Publisher struct {
	store *store.Store
}

// NewPublisher 构造快照发布器。
func NewPublisher(s *store.Store) *Publisher {
	return &Publisher{store: s}
}

// Payload 是快照内容的结构化载体。
type Payload struct {
	RunName        string   `json:"run_name"`
	BaselineMatrix string   `json:"baseline_matrix_id"`
	Frames         int      `json:"frames"`
	Candidates     int      `json:"candidates"`
	Attributions   []string `json:"attributions"`
	Summary        string   `json:"summary"`
}

// Create 为运行创建新版本快照（草稿态），返回快照实体。
// 版本号自动递增：当前最大版本 + 1。
func (p *Publisher) Create(ctx context.Context, runID, baselineMatrixID string, payload Payload) (*model.DiagnosisSnapshot, error) {
	latest, err := p.store.LatestSnapshotVersion(ctx, runID)
	if err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("快照内容序列化失败: %w", err)
	}
	snap := &model.DiagnosisSnapshot{
		ID:               fmt.Sprintf("snap_%s_v%d", runID, latest+1),
		RunID:            runID,
		Version:          latest + 1,
		BaselineMatrixID: baselineMatrixID,
		Content:          string(content),
		Status:           model.SnapshotStatusDraft,
	}
	if err := p.store.CreateSnapshot(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// Publish 把草稿快照发布；发布时若该运行已有已发布快照，则将其置为"替代"。
func (p *Publisher) Publish(ctx context.Context, snapID string) (*model.DiagnosisSnapshot, error) {
	snap, err := p.store.GetSnapshot(ctx, snapID)
	if err != nil {
		return nil, err
	}
	if err := model.CheckSnapshotTransition(snap.Status, model.SnapshotStatusPub); err != nil {
		return nil, err
	}
	// 同运行已发布过的旧快照全部标记为"替代"，保证同运行只保留一个已发布快照
	existing, err := p.store.ListSnapshotsByRun(ctx, snap.RunID)
	if err != nil {
		return nil, err
	}
	for _, s := range existing {
		if s.ID == snapID {
			continue
		}
		if s.Status != model.SnapshotStatusPub {
			continue
		}
		if err := p.store.UpdateSnapshotStatus(ctx, s.ID, model.SnapshotStatusReplaced); err != nil {
			return nil, err
		}
	}
	if err := p.store.UpdateSnapshotStatus(ctx, snapID, model.SnapshotStatusPub); err != nil {
		return nil, err
	}
	snap.Status = model.SnapshotStatusPub
	return snap, nil
}

// List 列出某运行的全部快照。
func (p *Publisher) List(ctx context.Context, runID string) ([]*model.DiagnosisSnapshot, error) {
	return p.store.ListSnapshotsByRun(ctx, runID)
}

// Get 按 ID 读取快照。
func (p *Publisher) Get(ctx context.Context, id string) (*model.DiagnosisSnapshot, error) {
	return p.store.GetSnapshot(ctx, id)
}
