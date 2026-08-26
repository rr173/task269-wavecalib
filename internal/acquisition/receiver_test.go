package acquisition

import (
	"testing"
)

func TestFrameInputValidate(t *testing.T) {
	valid := FrameInput{RunID: "r1", Seq: 1, TimestampMS: 100, StarID: "s1", MatrixID: "m1", Residuals: []float64{0.1, 0.2}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("合法输入被拒绝: %v", err)
	}
	bad := []FrameInput{
		{RunID: "", Seq: 1, TimestampMS: 100, StarID: "s1", MatrixID: "m1", Residuals: []float64{1}},
		{RunID: "r1", Seq: 0, TimestampMS: 100, StarID: "s1", MatrixID: "m1", Residuals: []float64{1}},
		{RunID: "r1", Seq: 1, TimestampMS: -5, StarID: "s1", MatrixID: "m1", Residuals: []float64{1}},
		{RunID: "r1", Seq: 1, TimestampMS: 100, StarID: "", MatrixID: "m1", Residuals: []float64{1}},
		{RunID: "r1", Seq: 1, TimestampMS: 100, StarID: "s1", MatrixID: "m1", Residuals: nil},
	}
	for i, in := range bad {
		if err := in.Validate(); err == nil {
			t.Fatalf("非法输入 #%d 未被拒绝: %+v", i, in)
		}
	}
}

func TestChecksumDeterministic(t *testing.T) {
	in := FrameInput{RunID: "r1", Seq: 1, TimestampMS: 100, StarID: "s1", MatrixID: "m1", Residuals: []float64{0.1, 0.2}}
	a, err := Checksum(&in)
	if err != nil {
		t.Fatalf("Checksum: %v", err)
	}
	b, err := Checksum(&in)
	if err != nil {
		t.Fatalf("Checksum: %v", err)
	}
	if a != b {
		t.Fatalf("Checksum 不稳定: %s vs %s", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("Checksum 长度 = %d, want 64", len(a))
	}
}
