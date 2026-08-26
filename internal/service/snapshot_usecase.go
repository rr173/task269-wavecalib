package service

import (
	"context"
	"fmt"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/snapshot"
)

// SnapshotUsecase 封装诊断快照的创建与发布用例。
type SnapshotUsecase struct {
	svc *Service
}

// CreateSnapshot 为运行创建诊断快照（草稿态），汇总当前诊断状态。
func (u *SnapshotUsecase) CreateSnapshot(ctx context.Context, runID, baselineMatrixID string) (*model.DiagnosisSnapshot, error) {
	run, err := u.svc.Store.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	frames, err := u.svc.Store.CountAnalyzableFramesByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	cands, err := u.svc.Store.ListDriftCandidatesByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	var attributions []string
	for _, c := range cands {
		attributions = append(attributions, fmt.Sprintf("%s(%.2f)", c.Attribution, c.Confidence))
	}
	payload := snapshot.Payload{
		RunName:        run.Name,
		BaselineMatrix: baselineMatrixID,
		Frames:         frames,
		Candidates:     len(cands),
		Attributions:   attributions,
		Summary:        summarize(run, cands),
	}
	return u.svc.Snapshot.Create(ctx, runID, baselineMatrixID, payload)
}

// PublishSnapshot 发布草稿快照。
func (u *SnapshotUsecase) PublishSnapshot(ctx context.Context, snapID string) (*model.DiagnosisSnapshot, error) {
	return u.svc.Snapshot.Publish(ctx, snapID)
}

// ListSnapshots 列出运行的全部快照。
func (u *SnapshotUsecase) ListSnapshots(ctx context.Context, runID string) ([]*model.DiagnosisSnapshot, error) {
	return u.svc.Snapshot.List(ctx, runID)
}

// GetSnapshot 按 ID 读取快照。
func (u *SnapshotUsecase) GetSnapshot(ctx context.Context, id string) (*model.DiagnosisSnapshot, error) {
	return u.svc.Snapshot.Get(ctx, id)
}

// summarize 生成快照的一句话总结。
func summarize(run *model.ObservationRun, cands []*model.DriftCandidate) string {
	calibCount, atmoCount, cmdCount, unknownCount := 0, 0, 0, 0
	for _, c := range cands {
		switch c.Attribution {
		case model.AttributionCalib:
			calibCount++
		case model.AttributionAtmo:
			atmoCount++
		case model.AttributionCommand:
			cmdCount++
		default:
			unknownCount++
		}
	}
	return fmt.Sprintf("运行 %s：校准相关 %d、大气相关 %d、命令相关 %d、证据不足 %d",
		run.Name, calibCount, atmoCount, cmdCount, unknownCount)
}
