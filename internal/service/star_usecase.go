package service

import (
	"context"

	"task269-wavecalib/internal/quality"
	"task269-wavecalib/internal/model"
)

// StarUsecase 封装参考星登记与质量评估用例。
type StarUsecase struct {
	svc *Service
}

// StarInput 是登记参考星的请求体。
type StarInput struct {
	Name                string  `json:"name"`
	SNR                 float64 `json:"snr"`
	PositionErrorArcsec float64 `json:"position_error_arcsec"`
}

// RegisterStar 登记参考星，并立即评估初始质量。
func (u *StarUsecase) RegisterStar(ctx context.Context, in StarInput) (*model.ReferenceStar, error) {
	if in.Name == "" {
		return nil, &model.ValidationError{Field: "name", Message: "不能为空"}
	}
	if in.SNR <= 0 {
		return nil, &model.ValidationError{Field: "snr", Message: "必须为正数"}
	}
	if in.PositionErrorArcsec < 0 {
		return nil, &model.ValidationError{Field: "position_error_arcsec", Message: "不能为负数"}
	}
	score := quality.Score(in.SNR, in.PositionErrorArcsec)
	st := &model.ReferenceStar{
		ID:                 NewID("star"),
		Name:               in.Name,
		SNR:                in.SNR,
		PositionErrorArcsec: in.PositionErrorArcsec,
		QualityScore:       score,
		Status:             model.StarStatusUsable,
	}
	// 用分类逻辑确定初始状态
	st.Status = quality.Classify(score, in.PositionErrorArcsec)
	if err := u.svc.Store.CreateReferenceStar(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

// GetStar 读取参考星。
func (u *StarUsecase) GetStar(ctx context.Context, id string) (*model.ReferenceStar, error) {
	return u.svc.Store.GetReferenceStar(ctx, id)
}

// ListStars 列出参考星。
func (u *StarUsecase) ListStars(ctx context.Context) ([]*model.ReferenceStar, error) {
	return u.svc.Store.ListReferenceStars(ctx)
}

// ReEvaluateStar 重新评估参考星质量（可能随观测条件变化）。
func (u *StarUsecase) ReEvaluateStar(ctx context.Context, id string) (*model.ReferenceStar, error) {
	score, status, err := u.svc.Quality.Evaluate(ctx, id)
	if err != nil {
		return nil, err
	}
	st, err := u.svc.Store.GetReferenceStar(ctx, id)
	if err != nil {
		return nil, err
	}
	st.QualityScore = score
	st.Status = status
	return st, nil
}

// MatrixUsecase 封装校准矩阵登记与解析用例。
type MatrixUsecase struct {
	svc *Service
}

// MatrixInput 是登记校准矩阵的请求体。
type MatrixInput struct {
	Version        string `json:"version"`
	Rows           int    `json:"rows"`
	Cols           int    `json:"cols"`
	EffectiveFrom  int64  `json:"effective_from_ms"`
	EffectiveUntil int64  `json:"effective_until_ms"`
}

// RegisterMatrix 登记校准矩阵版本。
func (u *MatrixUsecase) RegisterMatrix(ctx context.Context, in MatrixInput) (*model.CalibrationMatrix, error) {
	if in.Version == "" {
		return nil, &model.ValidationError{Field: "version", Message: "不能为空"}
	}
	mat := &model.CalibrationMatrix{
		ID:             NewID("mat"),
		Version:        in.Version,
		Rows:           in.Rows,
		Cols:           in.Cols,
		EffectiveFrom:  in.EffectiveFrom,
		EffectiveUntil: in.EffectiveUntil,
		Status:         model.MatrixStatusDraft,
	}
	if err := u.svc.Calibration.Register(ctx, mat); err != nil {
		return nil, err
	}
	return mat, nil
}

// GetMatrix 读取校准矩阵。
func (u *MatrixUsecase) GetMatrix(ctx context.Context, id string) (*model.CalibrationMatrix, error) {
	return u.svc.Store.GetCalibrationMatrix(ctx, id)
}

// ListMatrices 列出校准矩阵。
func (u *MatrixUsecase) ListMatrices(ctx context.Context) ([]*model.CalibrationMatrix, error) {
	return u.svc.Store.ListCalibrationMatrices(ctx)
}

// ResolveMatrixAt 解析时刻生效的校准矩阵。
func (u *MatrixUsecase) ResolveMatrixAt(ctx context.Context, ts int64) (*model.CalibrationMatrix, error) {
	return u.svc.Calibration.ResolveForTimestamp(ctx, ts)
}
