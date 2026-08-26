// Package diagnosis 实现波前残差的模式分解、漂移归因与诊断报告。
package diagnosis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// 残差模式名称常量（Zernike 前五阶的简化近似）。
const (
	ModeTiltX  = "tilt-x"
	ModeTiltY  = "tilt-y"
	ModeFocus  = "focus"
	ModeAstig  = "astig"
	ModeComa   = "coma"
)

// ModeNames 返回全部支持的模式名。
func ModeNames() []string {
	return []string{ModeTiltX, ModeTiltY, ModeFocus, ModeAstig, ModeComa}
}

// ResidualAnalyzer 对波前帧残差做模式分解与偏差分析。
type ResidualAnalyzer struct {
	store *store.Store
}

// NewResidualAnalyzer 构造残差分析器。
func NewResidualAnalyzer(s *store.Store) *ResidualAnalyzer {
	return &ResidualAnalyzer{store: s}
}

// RMS 计算残差向量的均方根。
func RMS(residuals []float64) float64 {
	if len(residuals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range residuals {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(residuals)))
}

// Decompose 将一维残差向量近似分解为五种模式系数。
//
// 方法：把残差向量按行优先视作 n x n 波前图（n 取最接近的平方根向下取整），
// 分别投影到倾斜 x、倾斜 y、离焦、像散、彗差五个基上，归一化后得到系数。
// 该近似面向离散波前传感器，用于漂移归因的"模式偏移"判断。
func Decompose(residuals []float64) map[string]float64 {
	n := int(math.Sqrt(float64(len(residuals))))
	if n < 2 {
		n = 2
	}
	coeff := make(map[string]float64)
	modes := ModeNames()
	for _, name := range modes {
		coeff[name] = project(residuals, n, basis(name, n))
	}
	return coeff
}

// project 计算残差在给定基上的归一化投影系数。
func project(residuals []float64, n int, basis []float64) float64 {
	if len(basis) == 0 {
		return 0
	}
	var dot, norm float64
	for i := 0; i < n*n && i < len(residuals); i++ {
		dot += residuals[i] * basis[i]
		norm += basis[i] * basis[i]
	}
	if norm == 0 {
		return 0
	}
	return dot / norm
}

// basis 生成第 i 行第 j 列处的模式基采样值（行优先展开）。
func basis(name string, n int) []float64 {
	out := make([]float64, n*n)
	mid := float64(n-1) / 2.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := float64(j) - mid
			y := float64(i) - mid
			r := math.Hypot(x, y)
			theta := math.Atan2(y, x)
			var v float64
			switch name {
			case ModeTiltX:
				v = x
			case ModeTiltY:
				v = y
			case ModeFocus:
				v = 2*r*r - 1
			case ModeAstig:
				v = r * r * math.Cos(2*theta)
			case ModeComa:
				v = (3*r*r - 2) * r * math.Cos(theta)
			}
			out[i*n+j] = v
		}
	}
	return out
}

// Deviation 计算单帧某模式的偏差量：|当前系数| - |基线系数|。
func Deviation(current, baseline float64) float64 {
	return math.Abs(current) - math.Abs(baseline)
}

// AnalyzeFrame 对单帧执行残差模式分解并持久化，返回各模式的偏差。
// baseline 为校准基线模式系数（通常取运行创建时生效矩阵对应帧的系数均值）。
func (a *ResidualAnalyzer) AnalyzeFrame(ctx context.Context, runID, frameID string, residuals []float64, baseline map[string]float64) (map[string]float64, error) {
	coeff := Decompose(residuals)
	dev := make(map[string]float64)
	for _, name := range ModeNames() {
		c := coeff[name]
		b := baseline[name]
		d := Deviation(c, b)
		dev[name] = d
		rm := &model.ResidualMode{
			ID:          fmt.Sprintf("rm_%s_%s", frameID, name),
			RunID:       runID,
			FrameID:     frameID,
			ModeName:    name,
			Coefficient: c,
			Deviation:   d,
		}
		if err := a.store.CreateResidualMode(ctx, rm); err != nil {
			return nil, err
		}
	}
	return dev, nil
}

// DominantDeviation 返回偏差绝对值最大的模式名及其偏差。
func DominantDeviation(dev map[string]float64) (string, float64) {
	names := ModeNames()
	sort.Slice(names, func(i, j int) bool {
		return math.Abs(dev[names[i]]) > math.Abs(dev[names[j]])
	})
	return names[0], dev[names[0]]
}

// FormatModes 输出模式系数摘要文本。
func FormatModes(coeff map[string]float64) string {
	var parts []string
	for _, name := range ModeNames() {
		parts = append(parts, fmt.Sprintf("%s=%.3f", name, coeff[name]))
	}
	return strings.Join(parts, ", ")
}
