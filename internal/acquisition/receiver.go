// Package acquisition 负责波前帧的采集接收：校验、幂等去重、落库。
package acquisition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// Receiver 是波前帧接收器，封装帧幂等写入逻辑。
type Receiver struct {
	store *store.Store
}

// NewReceiver 构造接收器。
func NewReceiver(s *store.Store) *Receiver {
	return &Receiver{store: s}
}

// FrameInput 是接收波前帧的请求体。
type FrameInput struct {
	RunID       string    `json:"run_id"`
	Seq         int64     `json:"seq"`
	TimestampMS int64     `json:"timestamp_ms"`
	StarID      string    `json:"star_id"`
	MatrixID    string    `json:"matrix_id"`
	Residuals   []float64 `json:"residuals"`
}

// Validate 校验帧入参的基本约束。
func (in *FrameInput) Validate() error {
	if strings.TrimSpace(in.RunID) == "" {
		return &model.ValidationError{Field: "run_id", Message: "不能为空"}
	}
	if in.Seq <= 0 {
		return &model.ValidationError{Field: "seq", Message: "必须为正整数"}
	}
	if in.TimestampMS <= 0 {
		return &model.ValidationError{Field: "timestamp_ms", Message: "必须为正整数"}
	}
	if strings.TrimSpace(in.StarID) == "" {
		return &model.ValidationError{Field: "star_id", Message: "不能为空"}
	}
	if strings.TrimSpace(in.MatrixID) == "" {
		return &model.ValidationError{Field: "matrix_id", Message: "不能为空"}
	}
	if len(in.Residuals) == 0 {
		return &model.ValidationError{Field: "residuals", Message: "不能为空"}
	}
	for i, v := range in.Residuals {
		if v != v || v == 0 && fmt.Sprintf("%.0f", v) == "NaN" {
			return &model.ValidationError{Field: fmt.Sprintf("residuals[%d]", i), Message: "包含 NaN"}
		}
	}
	return nil
}

// Checksum 计算帧内容的 SHA-256 校验和，用于幂等识别与内容防篡改。
func Checksum(in *FrameInput) (string, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// Ingest 接收一帧波前：校验 -> 幂等判断 -> 落库。
// 若 (run, seq) 已存在且校验和一致，返回已存在帧且无错误（幂等成功）；
// 若校验和不同则返回 ErrConflict（内容被篡改的重复投递）。
func (r *Receiver) Ingest(ctx context.Context, in *FrameInput) (*model.WavefrontFrame, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	// 校验引用的实体存在
	if _, err := r.store.GetRun(ctx, in.RunID); err != nil {
		return nil, fmt.Errorf("%w: 运行 %s", model.ErrInvalidInput, in.RunID)
	}
	if _, err := r.store.GetReferenceStar(ctx, in.StarID); err != nil {
		return nil, fmt.Errorf("%w: 参考星 %s", model.ErrInvalidInput, in.StarID)
	}
	if _, err := r.store.GetCalibrationMatrix(ctx, in.MatrixID); err != nil {
		return nil, fmt.Errorf("%w: 校准矩阵 %s", model.ErrInvalidInput, in.MatrixID)
	}

	cksum, err := Checksum(in)
	if err != nil {
		return nil, err
	}

	// 幂等去重：同一 (run, seq) 已存在时比较校验和
	existing, err := r.store.GetFrameBySeq(ctx, in.RunID, in.Seq)
	if err == nil {
		if existing.Checksum == cksum {
			return existing, nil // 重复投递，内容一致，幂等成功
		}
		return nil, fmt.Errorf("%w: 帧 (run=%s, seq=%d) 校验和冲突", model.ErrConflict, in.RunID, in.Seq)
	}

	frame := &model.WavefrontFrame{
		ID:          fmt.Sprintf("wf_%s_%d", in.RunID, in.Seq),
		RunID:       in.RunID,
		Seq:         in.Seq,
		TimestampMS: in.TimestampMS,
		StarID:      in.StarID,
		MatrixID:    in.MatrixID,
		Residuals:   in.Residuals,
		Status:      model.FrameStatusRaw,
		Checksum:    cksum,
	}
	if err := r.store.CreateFrame(ctx, frame); err != nil {
		return nil, err
	}
	return frame, nil
}
