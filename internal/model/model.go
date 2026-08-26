// Package model 定义自适应光学波前校准漂移诊断服务的核心实体与状态机。
//
// 领域对象：观测运行（ObservationRun）、波前帧（WavefrontFrame）、
// 参考星（ReferenceStar）、校准矩阵（CalibrationMatrix）、
// 残差模式（ResidualMode）、漂移候选（DriftCandidate）、诊断快照（DiagnosisSnapshot）。
package model

import (
	"time"
)

// 状态常量：观测运行生命周期。
const (
	RunStatusCollecting = "采集中" // 正在接收波前帧
	RunStatusAnalyzing  = "待分析" // 帧收齐，等待残差分析
	RunStatusReviewing  = "需复核" // 已生成漂移候选，等待工程师复核
	RunStatusConfirmed  = "确认"   // 漂移候选已确认
	RunStatusSealed     = "封存"   // 诊断结束，输入不可变
)

// 状态常量：波前帧生命周期。
const (
	FrameStatusRaw        = "原始"
	FrameStatusValid      = "有效"
	FrameStatusRefPoor    = "参考不足"
	FrameStatusAbnormal   = "异常"
	FrameStatusExcluded   = "排除"
)

// 状态常量：漂移候选归因。
const (
	AttributionAtmo     = "大气相关"
	AttributionCalib    = "校准相关"
	AttributionCommand  = "命令相关"
	AttributionUnknown  = "证据不足"
)

// 状态常量：诊断快照生命周期。
const (
	SnapshotStatusDraft   = "草稿"
	SnapshotStatusPub     = "发布"
	SnapshotStatusReplaced = "替代"
)

// 状态常量：参考星生命周期。
const (
	StarStatusUsable = "可用"
	StarStatusWeak   = "低质"
	StarStatusDead   = "废弃"
)

// 状态常量：校准矩阵生命周期。
const (
	MatrixStatusDraft    = "草稿"
	MatrixStatusActive   = "生效"
	MatrixStatusExpired  = "过期"
	MatrixStatusRetired  = "废止"
)

// ObservationRun 是一次望远镜观测运行，承载波前采集与漂移诊断全过程。
type ObservationRun struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TelescopeID string    `json:"telescope_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WavefrontFrame 是一帧波前传感器残差快照。
// 幂等键：同一 run 下 seq 唯一，重复投递按 checksum 校验后忽略。
type WavefrontFrame struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	Seq         int64     `json:"seq"`
	TimestampMS int64     `json:"timestamp_ms"`
	StarID      string    `json:"star_id"`
	MatrixID    string    `json:"matrix_id"`
	Residuals   []float64 `json:"residuals"` // 与参考波前的逐点残差（微米）
	Status      string    `json:"status"`
	Checksum    string    `json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReferenceStar 是波前传感器标定用的参考星。
// quality_score 由质量模块按 SNR 与定位误差综合得出，0~100。
type ReferenceStar struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	SNR                float64   `json:"snr"`
	PositionErrorArcsec float64  `json:"position_error_arcsec"`
	QualityScore       float64   `json:"quality_score"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}

// CalibrationMatrix 是变形镜补偿用的校准矩阵版本。
// 每个版本带生效区间，帧按时间戳匹配生效版本；过期版本会触发校准漂移告警。
type CalibrationMatrix struct {
	ID            string    `json:"id"`
	Version       string    `json:"version"`
	Rows          int       `json:"rows"`
	Cols          int       `json:"cols"`
	EffectiveFrom int64     `json:"effective_from_ms"`
	EffectiveUntil int64    `json:"effective_until_ms"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// ResidualMode 是单帧残差的模式分解结果。
// mode_name 取值：tilt-x/tilt-y/focus/astig/coma 之一；
// deviation 为相对校准基线的偏差量。
type ResidualMode struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	FrameID     string    `json:"frame_id"`
	ModeName    string    `json:"mode_name"`
	Coefficient float64   `json:"coefficient"`
	Deviation   float64   `json:"deviation"`
	CreatedAt   time.Time `json:"created_at"`
}

// DriftCandidate 是一次漂移归因结论，把异常帧关联到可能的原因类别。
type DriftCandidate struct {
	ID          string    `json:"id"`
	RunID       string    `json:"run_id"`
	FrameID     string    `json:"frame_id"`
	MatrixID    string    `json:"matrix_id"`
	Attribution string    `json:"attribution"`
	Confidence  float64   `json:"confidence"` // 0~1
	Detail      string    `json:"detail"`
	Status      string    `json:"status"` // 候选/确认/否决
	CreatedAt   time.Time `json:"created_at"`
}

// DiagnosisSnapshot 是冻结的诊断快照，锁定校准基线与漂移结论。
type DiagnosisSnapshot struct {
	ID                string    `json:"id"`
	RunID             string    `json:"run_id"`
	Version           int       `json:"version"`
	BaselineMatrixID  string    `json:"baseline_matrix_id"`
	Content           string    `json:"content"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}
