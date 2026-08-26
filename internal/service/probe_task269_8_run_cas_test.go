package service

import (
	"context"
	"testing"

	"task269-wavecalib/internal/model"
)

func TestRunStatusCASRejectsStaleExpected(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "cas-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Store.UpdateRunStatus(ctx, run.ID, model.RunStatusCollecting, model.RunStatusAnalyzing); err != nil {
		t.Fatal(err)
	}
	if err := svc.Store.UpdateRunStatus(ctx, run.ID, model.RunStatusCollecting, model.RunStatusSealed); err == nil {
		t.Fatal("stale expected status must be rejected")
	}
}
