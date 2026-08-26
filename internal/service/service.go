// Package service 编排采集、质量、校准、诊断与快照各模块，向 HTTP 层暴露用例级接口。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/calibration"
	"task269-wavecalib/internal/diagnosis"
	"task269-wavecalib/internal/quality"
	"task269-wavecalib/internal/snapshot"
	"task269-wavecalib/internal/store"
)

// Service 聚合所有业务模块，是 HTTP 层的唯一依赖。
type Service struct {
	Store       *store.Store
	Receiver    *acquisition.Receiver
	Quality     *quality.Scorer
	Calibration *calibration.Manager
	Diagnosis   *diagnosis.ResidualAnalyzer
	Attribution *diagnosis.AttributionEngine
	Snapshot    *snapshot.Publisher

	Runs      *RunUsecase
	Frames    *FrameUsecase
	Stars     *StarUsecase
	Matrices  *MatrixUsecase
	Diagnoses *DiagnosisUsecase
	Snaps     *SnapshotUsecase
}

// New 构造服务聚合。
func New(s *store.Store) *Service {
	cm := calibration.NewManager(s)
	svc := &Service{
		Store:       s,
		Receiver:    acquisition.NewReceiver(s),
		Quality:     quality.NewScorer(s),
		Calibration: cm,
		Diagnosis:   diagnosis.NewResidualAnalyzer(s),
		Attribution: diagnosis.NewAttributionEngine(s, cm),
		Snapshot:    snapshot.NewPublisher(s),
	}
	svc.Runs = &RunUsecase{svc: svc}
	svc.Frames = &FrameUsecase{svc: svc}
	svc.Stars = &StarUsecase{svc: svc}
	svc.Matrices = &MatrixUsecase{svc: svc}
	svc.Diagnoses = &DiagnosisUsecase{svc: svc}
	svc.Snaps = &SnapshotUsecase{svc: svc}
	return svc
}

// NewID 生成短随机 ID。
func NewID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// Context 便捷别名，统一 ctx 语义。
type Context = context.Context
