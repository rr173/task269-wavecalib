package diagnosis

import (
	"context"
	"fmt"
	"math"

	"task269-wavecalib/internal/calibration"
	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// 漂移归因判定阈值。
const (
	// DeviationThreshold 偏差超过该阈值（微米）视为显著模式偏移。
	DeviationThreshold = 0.5
	// StarScoreThreshold 参考星质量分低于该值时归因置信度下调。
	StarScoreThreshold = 60.0
	// ConfidenceBase 基础置信度。
	ConfidenceBase = 0.7
)

// AttributionEngine 执行漂移归因：结合残差偏差、校准矩阵时效与参考星质量，
// 把异常帧归因到大气/校准/命令/证据不足四类之一。
type AttributionEngine struct {
	store   *store.Store
	calib   *calibration.Manager
}

// NewAttributionEngine 构造归因引擎。
func NewAttributionEngine(s *store.Store, cm *calibration.Manager) *AttributionEngine {
	return &AttributionEngine{store: s, calib: cm}
}

// Input 是一次漂移归因的输入摘要。
type Input struct {
	RunID          string
	Frame          *model.WavefrontFrame
	ModesDeviation map[string]float64 // 模式偏差
	Star           *model.ReferenceStar
	Matrix         *model.CalibrationMatrix
	NowMS          int64
}

// Attribute 对单帧执行漂移归因，返回候选与置信度。
//
// 归因逻辑：
//  1. 矩阵已过期或年龄超过 MaxMatrixAgeDays -> 校准相关（置信度高）。
//  2. 显著偏差集中在离焦/彗差且参考星低质 -> 大气相关（视宁度退化）。
//  3. 显著偏差集中在倾斜模式 -> 命令相关（变形镜命令失配）。
//  4. 无显著偏差或信号矛盾 -> 证据不足。
func (e *AttributionEngine) Attribute(ctx context.Context, in *Input) (*model.DriftCandidate, error) {
	name, dev := DominantDeviation(in.ModesDeviation)
	confidence := ConfidenceBase
	detail := buildDetail(in, name, dev)

	var attribution string
	// 矩阵时效以帧采集时刻 (in.NowMS) 为基准判定：既看是否在生效区间内，
	// 也看过建议年龄。Status 在登记时按当时墙钟冻结，无法反映采集时刻，
	// 故过期与否由 EffectiveInterval 与采集时刻直接推导。
	matrixStale := in.Matrix != nil && (!calibration.IsActiveAt(in.Matrix, in.NowMS) || calibration.IsStale(in.Matrix, in.NowMS))
	switch {
	case matrixStale && math.Abs(dev) >= DeviationThreshold:
		attribution = model.AttributionCalib
		confidence = math.Min(0.95, confidence+0.15)
	case math.Abs(dev) >= DeviationThreshold && in.Star != nil && in.Star.QualityScore < StarScoreThreshold:
		attribution = model.AttributionAtmo
		confidence = math.Min(0.9, confidence+0.1)
	case math.Abs(dev) >= DeviationThreshold && (name == ModeTiltX || name == ModeTiltY):
		attribution = model.AttributionCommand
		confidence = math.Min(0.9, confidence+0.1)
	default:
		attribution = model.AttributionUnknown
		confidence = math.Max(0.3, confidence-0.3)
	}

	cand := &model.DriftCandidate{
		ID:          fmt.Sprintf("dc_%s_%s", in.Frame.ID, name),
		RunID:       in.RunID,
		FrameID:     in.Frame.ID,
		MatrixID:    in.Frame.MatrixID,
		Attribution: attribution,
		Confidence:  confidence,
		Detail:      detail,
		Status:      "候选",
	}
	if err := e.store.CreateDriftCandidate(ctx, cand); err != nil {
		return nil, err
	}
	return cand, nil
}

func buildDetail(in *Input, mode string, dev float64) string {
	starNote := "无参考星"
	if in.Star != nil {
		starNote = fmt.Sprintf("参考星质量分=%.1f", in.Star.QualityScore)
	}
	matrixNote := "无矩阵"
	if in.Matrix != nil {
		matrixNote = calibration.EffectiveIntervalSummary(in.Matrix)
		if !calibration.IsActiveAt(in.Matrix, in.NowMS) {
			matrixNote += "（采集时刻已超出生效区间，校准漂移风险）"
		} else if calibration.IsStale(in.Matrix, in.NowMS) {
			matrixNote += "（已过建议年龄，有漂移风险）"
		}
	}
	return fmt.Sprintf("帧 %s 主导模式 %s 偏差 %.2f 微米；%s；%s",
		in.Frame.ID, mode, dev, starNote, matrixNote)
}
