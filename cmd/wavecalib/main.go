// wavecalib 自适应光学波前校准漂移诊断服务入口。
//
// 入口契约：
//
//	--addr :8080        HTTP 监听地址
//	--db <path>         SQLite 数据库路径（默认 wavecalib.db）
//	--smoke-test        自检模式：真实创建运行/参考星/校准矩阵/波前帧，
//	                    执行残差分析、漂移归因与快照发布闭环，
//	                    关闭并重新打开数据库验证持久化与重启恢复，
//	                    全程以 0 退出码结束。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/service"
	"task269-wavecalib/internal/store"
	"task269-wavecalib/internal/httpapi"
)

func main() {
	addr := flag.String("addr", ":8080", "http listen address")
	dbPath := flag.String("db", "wavecalib.db", "sqlite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end self test and exit")
	flag.Parse()

	logger := log.New(os.Stdout, "[wavecalib] ", log.LstdFlags)

	if *smoke {
		if err := runSmokeTest(*dbPath, logger); err != nil {
			logger.Printf("SMOKE TEST FAILED: %v", err)
			os.Exit(1)
		}
		logger.Printf("SMOKE TEST PASSED")
		return
	}

	s, err := store.Open(*dbPath)
	if err != nil {
		logger.Fatalf("open store: %v", err)
	}
	defer s.Close()

	svc := service.New(s)
	h := httpapi.NewHandler(svc, logger)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           h.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Printf("listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("serve: %v", err)
	}
}

// runSmokeTest 执行端到端自检。
func runSmokeTest(dbPath string, logger *log.Logger) error {
	ctx := context.Background()

	// 第一遍：写入阶段
	s1, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("首次打开数据库: %w", err)
	}
	svc := service.New(s1)

	// 1. 创建观测运行
	run, err := svc.Runs.CreateRun(ctx, service.RunInput{Name: "smoke-run", TelescopeID: "tel-1"})
	if err != nil {
		s1.Close()
		return fmt.Errorf("创建运行: %w", err)
	}
	logger.Printf("创建运行 %s (状态=%s)", run.ID, run.Status)

	// 2. 登记参考星
	star, err := svc.Stars.RegisterStar(ctx, service.StarInput{Name: "ref-alpha", SNR: 120, PositionErrorArcsec: 0.15})
	if err != nil {
		s1.Close()
		return fmt.Errorf("登记参考星: %w", err)
	}
	logger.Printf("登记参考星 %s (质量分=%.1f, 状态=%s)", star.ID, star.QualityScore, star.Status)

	// 3. 登记校准矩阵（生效区间覆盖当前时间，未过期）
	nowMS := time.Now().UnixMilli()
	mat, err := svc.Matrices.RegisterMatrix(ctx, service.MatrixInput{
		Version:        "v1.0",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  nowMS - 3600_000,
		EffectiveUntil: nowMS + 86400_000,
	})
	if err != nil {
		s1.Close()
		return fmt.Errorf("登记校准矩阵: %w", err)
	}
	logger.Printf("登记校准矩阵 %s (版本=%s, 状态=%s)", mat.ID, mat.Version, mat.Status)

	// 4. 接收波前帧（3 帧）
	for i := 1; i <= 3; i++ {
		frame, err := svc.Frames.IngestFrame(ctx, newFrameInput(run.ID, star.ID, mat.ID, int64(i), nowMS))
		if err != nil {
			s1.Close()
			return fmt.Errorf("接收帧 %d: %w", i, err)
		}
		logger.Printf("接收帧 %s (seq=%d, 状态=%s)", frame.ID, frame.Seq, frame.Status)
	}

	// 5. 幂等重投：同一帧重复投递应返回已存在帧且不报错
	dup, err := svc.Frames.IngestFrame(ctx, newFrameInput(run.ID, star.ID, mat.ID, 1, nowMS))
	if err != nil {
		s1.Close()
		return fmt.Errorf("幂等重投失败: %w", err)
	}
	if dup.Seq != 1 {
		s1.Close()
		return fmt.Errorf("幂等重投返回了错误帧 seq=%d", dup.Seq)
	}
	logger.Printf("幂等重投通过 (帧 %s 未重复写入)", dup.ID)

	// 6. 推进状态：采集中 -> 待分析 -> 需复核
	if _, err := svc.Runs.AnalyzeRun(ctx, run.ID); err != nil {
		s1.Close()
		return fmt.Errorf("推进待分析: %w", err)
	}
	if _, err := svc.Runs.TransitionRun(ctx, run.ID, model.RunStatusReviewing); err != nil {
		s1.Close()
		return fmt.Errorf("推进需复核: %w", err)
	}

	// 7. 残差模式分析
	res, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	if err != nil {
		s1.Close()
		return fmt.Errorf("残差分析: %w", err)
	}
	if res.AnalyzedFrames != 3 {
		s1.Close()
		return fmt.Errorf("残差分析帧数异常: got %d want 3", res.AnalyzedFrames)
	}
	logger.Printf("残差分析完成：%d 帧, %d 条模式", res.AnalyzedFrames, len(res.Modes))

	// 8. 漂移归因
	cands, err := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	if err != nil {
		s1.Close()
		return fmt.Errorf("漂移归因: %w", err)
	}
	if len(cands) != 3 {
		s1.Close()
		return fmt.Errorf("漂移候选数量异常: got %d want 3", len(cands))
	}
	logger.Printf("漂移归因完成：%d 条候选", len(cands))

	// 9. 确认第一条候选
	if _, err := svc.Diagnoses.ConfirmCandidate(ctx, cands[0].ID); err != nil {
		s1.Close()
		return fmt.Errorf("确认候选: %w", err)
	}

	// 10. 创建并发布诊断快照
	snap, err := svc.Snaps.CreateSnapshot(ctx, run.ID, mat.ID)
	if err != nil {
		s1.Close()
		return fmt.Errorf("创建快照: %w", err)
	}
	pub, err := svc.Snaps.PublishSnapshot(ctx, snap.ID)
	if err != nil {
		s1.Close()
		return fmt.Errorf("发布快照: %w", err)
	}
	if pub.Status != model.SnapshotStatusPub {
		s1.Close()
		return fmt.Errorf("快照状态异常: got %s", pub.Status)
	}
	logger.Printf("快照 %s v%d 已发布", pub.ID, pub.Version)

	// 关闭第一遍连接，模拟进程重启
	if err := s1.Close(); err != nil {
		return fmt.Errorf("关闭数据库: %w", err)
	}

	// 第二遍：重开验证持久化与重启恢复
	s2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("重启后重新打开数据库: %w", err)
	}
	defer s2.Close()
	svc2 := service.New(s2)

	run2, err := svc2.Runs.GetRun(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("重启后读取运行: %w", err)
	}
	if run2.Status != model.RunStatusReviewing {
		return fmt.Errorf("重启后运行状态不一致: got %s", run2.Status)
	}
	frames2, err := svc2.Frames.ListFrames(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("重启后读取帧: %w", err)
	}
	if len(frames2) != 3 {
		return fmt.Errorf("重启后帧数不一致: got %d want 3", len(frames2))
	}
	snaps2, err := svc2.Snaps.ListSnapshots(ctx, run.ID)
	if err != nil {
		return fmt.Errorf("重启后读取快照: %w", err)
	}
	if len(snaps2) != 1 || snaps2[0].Status != model.SnapshotStatusPub {
		return fmt.Errorf("重启后快照状态不一致: got %d 条, 最新状态 %s", len(snaps2), snaps2[0].Status)
	}
	logger.Printf("重启恢复验证通过：运行=%s, 帧=%d, 快照=%s",
		run2.Status, len(frames2), snaps2[0].Status)

	return nil
}

// newFrameInput 构造带轻微模式偏移的测试帧（第 3 帧 tilt-x 偏差显著）。
func newFrameInput(runID, starID, matrixID string, seq, baseMS int64) *acquisition.FrameInput {
	n := 16
	residuals := make([]float64, n*n)
	mid := float64(n-1) / 2.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := float64(j) - mid
			y := float64(i) - mid
			v := 0.01*x + 0.02*y + 0.005*(x*x+y*y)
			if seq == 3 {
				v += 1.2 * x // 注入显著的 tilt-x 漂移
			}
			residuals[i*n+j] = v
		}
	}
	return &acquisition.FrameInput{
		RunID:       runID,
		Seq:         seq,
		TimestampMS: baseMS + seq*100,
		StarID:      starID,
		MatrixID:    matrixID,
		Residuals:   residuals,
	}
}
