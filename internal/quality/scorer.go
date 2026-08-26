// Package quality 评估参考星质量：SNR 与定位误差 -> 综合质量分与状态。
package quality

import (
	"context"
	"fmt"
	"math"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// 质量分阈值常量。
const (
	// ScoreThresholdGood 质量分高于此值视为可用参考星。
	ScoreThresholdGood = 70.0
	// ScoreThresholdWeak 质量分低于此值但高于废弃阈值视为低质。
	ScoreThresholdWeak = 40.0
	// MaxPositionErrorArcsec 定位误差超过此值直接废弃。
	MaxPositionErrorArcsec = 2.0
)

// Scorer 对参考星质量进行评分并更新状态。
type Scorer struct {
	store *store.Store
}

// NewScorer 构造质量评分器。
func NewScorer(s *store.Store) *Scorer {
	return &Scorer{store: s}
}

// Score 根据 SNR 与定位误差计算质量分（0~100）。
//
// 评分模型：
//   - SNR 贡献：SNR >= 100 时取满 60 分，低于 10 时趋近 0，中间线性。
//   - 定位误差贡献：误差 <= 0.2 弧秒取满 40 分，>= 2.0 弧秒取 0，中间线性。
func Score(snr, positionErrorArcsec float64) float64 {
	snrPart := 60.0 * clamp01((snr-10.0)/90.0)
	posPart := 40.0 * clamp01((MaxPositionErrorArcsec-positionErrorArcsec)/(MaxPositionErrorArcsec-0.2))
	return snrPart + posPart
}

// Classify 由质量分与定位误差得到参考星状态。
func Classify(score, positionErrorArcsec float64) string {
	if positionErrorArcsec > MaxPositionErrorArcsec {
		return model.StarStatusDead
	}
	switch {
	case score >= ScoreThresholdGood:
		return model.StarStatusUsable
	case score >= ScoreThresholdWeak:
		return model.StarStatusWeak
	default:
		return model.StarStatusDead
	}
}

// Evaluate 评估并持久化一颗参考星的质量，返回更新后的质量分与状态。
func (sc *Scorer) Evaluate(ctx context.Context, id string) (float64, string, error) {
	st, err := sc.store.GetReferenceStar(ctx, id)
	if err != nil {
		return 0, "", err
	}
	score := Score(st.SNR, st.PositionErrorArcsec)
	status := Classify(score, st.PositionErrorArcsec)
	if err := sc.store.UpdateStarQuality(ctx, id, score, status); err != nil {
		return 0, "", err
	}
	return score, status, nil
}

// Describe 输出质量评估的一句话描述（供诊断详情引用）。
func Describe(snr, posErr, score float64) string {
	status := Classify(score, posErr)
	return fmt.Sprintf("参考星 SNR=%.1f, 定位误差=%.2f\", 质量分=%.1f, 状态=%s",
		snr, posErr, score, status)
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
