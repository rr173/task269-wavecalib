package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

func TestPublishSnapshotReplacesPreviousPublished(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "pub-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatal(err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "pub-star", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	nowMS := time.Now().UnixMilli()
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version: "v1", Rows: 16, Cols: 16,
		EffectiveFrom: nowMS - 3600_000, EffectiveUntil: nowMS + 86400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
		RunID: run.ID, Seq: 1, TimestampMS: nowMS + 100,
		StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(1),
	}); err != nil {
		t.Fatal(err)
	}
	s1, err := svc.Snaps.CreateSnapshot(ctx, run.ID, mat.ID)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := svc.Snaps.CreateSnapshot(ctx, run.ID, mat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Snaps.PublishSnapshot(ctx, s1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Snaps.PublishSnapshot(ctx, s2.ID); err != nil {
		t.Fatal(err)
	}
	snaps, err := svc.Snaps.ListSnapshots(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	pub, replaced := 0, 0
	for _, s := range snaps {
		switch s.Status {
		case model.SnapshotStatusPub:
			pub++
		case model.SnapshotStatusReplaced:
			replaced++
		}
	}
	if pub != 1 || replaced != 1 {
		t.Fatalf("published=%d replaced=%d, want 1/1", pub, replaced)
	}
}
