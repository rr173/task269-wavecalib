package service

import (
	"context"
	"fmt"
	"time"

	"task269-wavecalib/internal/diagnosis"
	"task269-wavecalib/internal/model"
)

// DiagnosisUsecase 封装残差分析、漂移归因与候选管理的用例。
type DiagnosisUsecase struct {
	svc *Service
}

// ResidualResult 是一次残差分析运行的汇总结果。
type ResidualResult struct {
	RunID          string             `json:"run_id"`
	AnalyzedFrames int                `json:"analyzed_frames"`
	Modes          []*model.ResidualMode `json:"modes"`
}

// AnalyzeRunResiduals 对运行下全部有效帧执行残差模式分解。
// baseline 从运行的第一帧分解系数推导（作为校准基线）。
// 支持重复分析：每次运行前先清理该运行上一次的残差模式结果，
// 避免 (frame_id, mode_name) 唯一约束冲突，使再次分析可成功完成。
func (u *DiagnosisUsecase) AnalyzeRunResiduals(ctx context.Context, runID string) (*ResidualResult, error) {
	frames, err := u.svc.Store.ListFramesByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	// 基线：第一帧系数
	var baseline map[string]float64
	for _, f := range frames {
		if f.Status == model.FrameStatusExcluded {
			continue
		}
		baseline = diagnosis.Decompose(f.Residuals)
		break
	}
	if baseline == nil {
		return nil, fmt.Errorf("%w: 运行 %s 无可用帧，无法建立基线", model.ErrInvalidInput, runID)
	}
	// 清理上一次的残差模式结果，使重复分析成为覆盖式重算而非追加。
	if err := u.svc.Store.DeleteResidualModesByRun(ctx, runID); err != nil {
		return nil, err
	}
	result := &ResidualResult{RunID: runID}
	for _, f := range frames {
		if f.Status == model.FrameStatusExcluded {
			continue
		}
		dev, err := u.svc.Diagnosis.AnalyzeFrame(ctx, runID, f.ID, f.Residuals, baseline)
		if err != nil {
			return nil, err
		}
		_ = dev
		result.AnalyzedFrames++
	}
	modes, err := u.svc.Store.ListResidualModesByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	result.Modes = modes
	return result, nil
}

// AttributeDrift 对运行下全部有效帧执行漂移归因，生成候选列表。
func (u *DiagnosisUsecase) AttributeDrift(ctx context.Context, runID string) ([]*model.DriftCandidate, error) {
	frames, err := u.svc.Store.ListFramesByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	nowMS := time.Now().UnixMilli()
	var out []*model.DriftCandidate
	for _, f := range frames {
		if f.Status == model.FrameStatusExcluded {
			continue
		}
		modes, err := u.svc.Store.ListResidualModesByRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		dev := make(map[string]float64)
		for _, m := range modes {
			if m.FrameID == f.ID {
				dev[m.ModeName] = m.Deviation
			}
		}
		star, err := u.svc.Store.GetReferenceStar(ctx, f.StarID)
		if err != nil {
			return nil, err
		}
		matrix, err := u.svc.Store.GetCalibrationMatrix(ctx, f.MatrixID)
		if err != nil {
			return nil, err
		}
		frameMS := f.TimestampMS
		if frameMS <= 0 {
			frameMS = nowMS
		}
		cand, err := u.svc.Attribution.Attribute(ctx, &diagnosis.Input{
			RunID:          runID,
			Frame:          f,
			ModesDeviation: dev,
			Star:           star,
			Matrix:         matrix,
			NowMS:          frameMS,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, cand)
	}
	return out, nil
}

// ListCandidates 列出运行的全部漂移候选。
func (u *DiagnosisUsecase) ListCandidates(ctx context.Context, runID string) ([]*model.DriftCandidate, error) {
	return u.svc.Store.ListDriftCandidatesByRun(ctx, runID)
}

// ConfirmCandidate 确认漂移候选。
func (u *DiagnosisUsecase) ConfirmCandidate(ctx context.Context, id string) (*model.DriftCandidate, error) {
	c, err := u.svc.Store.GetDriftCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.svc.Store.UpdateDriftCandidateStatus(ctx, id, "已确认"); err != nil {
		return nil, err
	}
	c.Status = "已确认"
	return c, nil
}

// RejectCandidate 否决漂移候选。
func (u *DiagnosisUsecase) RejectCandidate(ctx context.Context, id string) (*model.DriftCandidate, error) {
	c, err := u.svc.Store.GetDriftCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.svc.Store.UpdateDriftCandidateStatus(ctx, id, "已否决"); err != nil {
		return nil, err
	}
	c.Status = "已否决"
	return c, nil
}
