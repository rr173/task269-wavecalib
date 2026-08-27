// Package calibration 管理校准矩阵版本：登记、时效判定、匹配与漂移关联。
package calibration

import (
	"context"
	"fmt"
	"time"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// MaxMatrixAgeDays 校准矩阵超过该年龄视为"过期风险"（触发校准漂移告警的阈值）。
const MaxMatrixAgeDays = 30

// Manager 是校准矩阵版本管理服务。
type Manager struct {
	store *store.Store
}

// NewManager 构造校准矩阵管理器。
func NewManager(s *store.Store) *Manager {
	return &Manager{store: s}
}

// Register 登记一个新校准矩阵版本；自动根据生效区间设置初始状态。
func (m *Manager) Register(ctx context.Context, mat *model.CalibrationMatrix) error {
	if mat.Rows <= 0 || mat.Cols <= 0 {
		return &model.ValidationError{Field: "rows/cols", Message: "矩阵维度必须为正"}
	}
	if mat.EffectiveFrom <= 0 {
		return &model.ValidationError{Field: "effective_from", Message: "生效时间必须为正"}
	}
	if mat.EffectiveUntil <= mat.EffectiveFrom {
		return &model.ValidationError{Field: "effective_until", Message: "失效时间必须晚于生效时间"}
	}
	if mat.Version == "" {
		return &model.ValidationError{Field: "version", Message: "版本号不能为空"}
	}
	now := time.Now().UnixMilli()
	switch {
	case now < mat.EffectiveFrom:
		mat.Status = model.MatrixStatusDraft
	case now >= mat.EffectiveFrom && now < mat.EffectiveUntil:
		mat.Status = model.MatrixStatusActive
	default:
		mat.Status = model.MatrixStatusExpired
	}
	return m.store.CreateCalibrationMatrix(ctx, mat)
}

// ResolveForTimestamp 为指定时刻解析应生效的校准矩阵版本。
// 若无生效矩阵则返回 ErrNotFound（此时帧应被标记为"参考不足"）。
func (m *Manager) ResolveForTimestamp(ctx context.Context, ts int64) (*model.CalibrationMatrix, error) {
	mat, err := m.store.ActiveMatrixAt(ctx, ts)
	if err != nil {
		return nil, fmt.Errorf("%w: 时刻 %d 无生效校准矩阵", model.ErrNotFound, ts)
	}
	return mat, nil
}

// IsActiveAt 判断矩阵在指定时刻是否处于生效区间（effective_from <= ts < effective_until）。
// 用于按帧采集时刻匹配矩阵版本，而非依赖登记时按墙钟冻结的 Status。
func IsActiveAt(mat *model.CalibrationMatrix, ts int64) bool {
	return mat.EffectiveFrom <= ts && ts < mat.EffectiveUntil
}

// AgeDays 计算矩阵从生效时刻至今的天数。
func AgeDays(mat *model.CalibrationMatrix, nowMS int64) int {
	age := nowMS - mat.EffectiveFrom
	if age < 0 {
		return 0
	}
	return int(age / (24 * 3600 * 1000))
}

// IsStale 判断矩阵是否已过最大建议年龄（校准漂移风险信号之一）。
func IsStale(mat *model.CalibrationMatrix, nowMS int64) bool {
	return AgeDays(mat, nowMS) > MaxMatrixAgeDays
}

// EffectiveIntervalSummary 输出矩阵生效区间摘要，用于诊断详情。
func EffectiveIntervalSummary(mat *model.CalibrationMatrix) string {
	from := time.UnixMilli(mat.EffectiveFrom).UTC().Format("2006-01-02 15:04:05")
	until := time.UnixMilli(mat.EffectiveUntil).UTC().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("版本 %s 生效区间 [%s, %s)", mat.Version, from, until)
}
