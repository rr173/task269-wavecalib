package diagnosis

import (
	"math"
	"testing"
)

func TestRMS(t *testing.T) {
	cases := []struct {
		name  string
		input []float64
		want  float64
	}{
		{"empty", nil, 0},
		{"zeros", []float64{0, 0, 0, 0}, 0},
		{"constant", []float64{3, 3, 3, 3}, 3},
		{"mixed", []float64{1, -1, 1, -1}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RMS(c.input)
			if math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("RMS(%v) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}

func TestDecomposeDeterministic(t *testing.T) {
	// 相同输入两次分解结果一致（确定性）
	a := Decompose([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
	b := Decompose([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
	for _, name := range ModeNames() {
		if math.Abs(a[name]-b[name]) > 1e-9 {
			t.Fatalf("模式 %s 分解不确定: %v vs %v", name, a[name], b[name])
		}
	}
}

func TestDecomposeTiltXDominant(t *testing.T) {
	// 纯 x 线性残差应使 tilt-x 系数绝对值最大
	n := 8
	residuals := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			residuals[i*n+j] = 1.5 * float64(j)
		}
	}
	coeff := Decompose(residuals)
	maxName, maxAbs := "", 0.0
	for _, name := range ModeNames() {
		if math.Abs(coeff[name]) > maxAbs {
			maxAbs = math.Abs(coeff[name])
			maxName = name
		}
	}
	if maxName != ModeTiltX {
		t.Fatalf("主导模式 = %s, want %s (coeff=%v)", maxName, ModeTiltX, coeff)
	}
}

func TestDeviation(t *testing.T) {
	if got := Deviation(1.0, 0.5); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("Deviation(1.0, 0.5) = %v, want 0.5", got)
	}
	if got := Deviation(-1.0, 0.5); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("Deviation(-1.0, 0.5) = %v, want 0.5", got)
	}
	if got := Deviation(0.3, 0.8); math.Abs(got+0.5) > 1e-9 {
		t.Fatalf("Deviation(0.3, 0.8) = %v, want -0.5", got)
	}
}
