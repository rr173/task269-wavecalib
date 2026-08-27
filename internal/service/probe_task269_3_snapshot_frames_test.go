package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/snapshot"
)

func TestSnapshotFrameCountExcludesMarkedFrames(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "snap-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatal(err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "snap-star", SNR: 120, PositionErrorArcsec: 0.1})
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
	var midFrame string
	for seq := int64(1); seq <= 3; seq++ {
		f, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
			RunID: run.ID, Seq: seq, TimestampMS: nowMS + seq*100,
			StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(seq),
		})
		if err != nil {
			t.Fatal(err)
		}
		if seq == 2 {
			midFrame = f.ID
		}
	}
	if _, err := svc.Frames.ExcludeFrame(ctx, midFrame); err != nil {
		t.Fatal(err)
	}
	snap, err := svc.Snaps.CreateSnapshot(ctx, run.ID, mat.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload snapshot.Payload
	if err := json.Unmarshal([]byte(snap.Content), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Frames != 2 {
		t.Fatalf("snapshot frames = %d, want 2 after exclude", payload.Frames)
	}
}
