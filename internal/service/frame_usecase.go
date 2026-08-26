package service

import (
	"context"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// FrameUsecase 封装波前帧接收与管理的用例。
type FrameUsecase struct {
	svc *Service
}

// IngestFrame 接收一帧波前（幂等）。
func (u *FrameUsecase) IngestFrame(ctx context.Context, in *acquisition.FrameInput) (*model.WavefrontFrame, error) {
	return u.svc.Receiver.Ingest(ctx, in)
}

// GetFrame 读取单帧。
func (u *FrameUsecase) GetFrame(ctx context.Context, id string) (*model.WavefrontFrame, error) {
	return u.svc.Store.GetFrame(ctx, id)
}

// ListFrames 列出某运行的帧。
func (u *FrameUsecase) ListFrames(ctx context.Context, runID string) ([]*model.WavefrontFrame, error) {
	return u.svc.Store.ListFramesByRun(ctx, runID)
}

// ExcludeFrame 把帧标记为"排除"（隔离失真帧）。
func (u *FrameUsecase) ExcludeFrame(ctx context.Context, id string) (*model.WavefrontFrame, error) {
	f, err := u.svc.Store.GetFrame(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.svc.Store.UpdateFrameStatus(ctx, id, model.FrameStatusExcluded); err != nil {
		return nil, err
	}
	f.Status = model.FrameStatusExcluded
	return f, nil
}

// RestoreFrame 把帧从"排除"恢复为"有效"。
func (u *FrameUsecase) RestoreFrame(ctx context.Context, id string) (*model.WavefrontFrame, error) {
	f, err := u.svc.Store.GetFrame(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.svc.Store.UpdateFrameStatus(ctx, id, model.FrameStatusValid); err != nil {
		return nil, err
	}
	f.Status = model.FrameStatusValid
	return f, nil
}
