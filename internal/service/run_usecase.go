package service

import (
	"context"
	"time"

	"task269-wavecalib/internal/model"
)

// RunUsecase 封装观测运行的生命周期用例。
type RunUsecase struct {
	svc *Service
}

// RunInput 是创建运行的请求体。
type RunInput struct {
	Name        string `json:"name"`
	TelescopeID string `json:"telescope_id"`
}

// CreateRun 创建运行，初始状态为"采集中"。
func (u *RunUsecase) CreateRun(ctx context.Context, in RunInput) (*model.ObservationRun, error) {
	if in.Name == "" {
		return nil, &model.ValidationError{Field: "name", Message: "不能为空"}
	}
	if in.TelescopeID == "" {
		return nil, &model.ValidationError{Field: "telescope_id", Message: "不能为空"}
	}
	r := &model.ObservationRun{
		ID:          NewID("run"),
		Name:        in.Name,
		TelescopeID: in.TelescopeID,
		Status:      model.RunStatusCollecting,
	}
	if err := u.svc.Store.CreateRun(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// GetRun 读取运行。
func (u *RunUsecase) GetRun(ctx context.Context, id string) (*model.ObservationRun, error) {
	return u.svc.Store.GetRun(ctx, id)
}

// ListRuns 按状态过滤列出运行。
func (u *RunUsecase) ListRuns(ctx context.Context, status string) ([]*model.ObservationRun, error) {
	return u.svc.Store.ListRuns(ctx, status)
}

// TransitionRun 执行运行状态流转；流转前读取当前状态做 CAS 校验。
func (u *RunUsecase) TransitionRun(ctx context.Context, id, next string) (*model.ObservationRun, error) {
	run, err := u.svc.Store.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := model.CheckRunTransition(run.Status, next); err != nil {
		return nil, err
	}
	if err := u.svc.Store.UpdateRunStatus(ctx, id, run.Status, next); err != nil {
		return nil, err
	}
	run.Status = next
	run.UpdatedAt = time.Now().UTC()
	return run, nil
}

// SealRun 封存运行：终态，输入不可变。
func (u *RunUsecase) SealRun(ctx context.Context, id string) (*model.ObservationRun, error) {
	return u.TransitionRun(ctx, id, model.RunStatusSealed)
}

// AnalyzeRun 把运行推进到"待分析"（采集完成，等待残差分析）。
func (u *RunUsecase) AnalyzeRun(ctx context.Context, id string) (*model.ObservationRun, error) {
	return u.TransitionRun(ctx, id, model.RunStatusAnalyzing)
}
