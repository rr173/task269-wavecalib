package service

import (
	"path/filepath"
	"testing"

	"task269-wavecalib/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "svc.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st)
}

func makeProbeResiduals(seq int64) []float64 {
	n := 16
	residuals := make([]float64, n*n)
	mid := float64(n-1) / 2.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := float64(j) - mid
			y := float64(i) - mid
			v := 0.01*x + 0.02*y + 0.005*(x*x+y*y)
			if seq == 3 {
				v += 1.2 * x
			}
			residuals[i*n+j] = v
		}
	}
	return residuals
}
