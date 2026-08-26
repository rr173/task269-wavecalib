package quality

import (
	"math"
	"testing"

	"task269-wavecalib/internal/model"
)

func TestScore(t *testing.T) {
	cases := []struct {
		name string
		snr  float64
		pos  float64
		want float64
	}{
		{"high snr low err", 150, 0.1, 100},
		{"low snr high err", 5, 3.0, 0},
		{"mid", 60, 0.5, 60.0*((60-10)/90.0) + 40.0*((2.0-0.5)/(2.0-0.2))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Score(c.snr, c.pos)
			if math.Abs(got-c.want) > 1e-6 {
				t.Fatalf("Score(%v, %v) = %v, want %v", c.snr, c.pos, got, c.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	if got := Classify(90, 0.1); got != model.StarStatusUsable {
		t.Fatalf("Classify(90, 0.1) = %v, want 可用", got)
	}
	if got := Classify(50, 0.5); got != model.StarStatusWeak {
		t.Fatalf("Classify(50, 0.5) = %v, want 低质", got)
	}
	if got := Classify(30, 0.5); got != model.StarStatusDead {
		t.Fatalf("Classify(30, 0.5) = %v, want 废弃", got)
	}
	if got := Classify(90, 2.5); got != model.StarStatusDead {
		t.Fatalf("Classify(90, 2.5) = %v, want 废弃（定位误差超限）", got)
	}
}
