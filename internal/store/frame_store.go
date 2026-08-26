package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"task269-wavecalib/internal/model"
)

// frameRow 是波前帧表的行结构。
type frameRow struct {
	id          string
	runID       string
	seq         int64
	timestampMS int64
	starID      string
	matrixID    string
	residuals   string
	status      string
	checksum    string
	createdAt   string
}

// CreateFrame 插入波前帧；同一 run 下 seq 重复时返回 ErrConflict（幂等键冲突）。
func (s *Store) CreateFrame(ctx context.Context, f *model.WavefrontFrame) error {
	res, err := json.Marshal(f.Residuals)
	if err != nil {
		return fmt.Errorf("%w: 残差序列化失败", model.ErrInvalidInput)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO wavefront_frames (id, run_id, seq, timestamp_ms, star_id, matrix_id, residuals, status, checksum, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.RunID, f.Seq, f.TimestampMS, f.StarID, f.MatrixID, string(res), f.Status, f.Checksum, now())
	if err != nil {
		return mapErr(err, "波前帧")
	}
	f.CreatedAt = parseTime(now())
	return nil
}

// GetFrame 按 ID 读取波前帧。
func (s *Store) GetFrame(ctx context.Context, id string) (*model.WavefrontFrame, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, run_id, seq, timestamp_ms, star_id, matrix_id, residuals, status, checksum, created_at
		 FROM wavefront_frames WHERE id = ?`, id)
	return scanFrame(row)
}

// GetFrameBySeq 按 (runID, seq) 读取帧，供幂等去重判断。
func (s *Store) GetFrameBySeq(ctx context.Context, runID string, seq int64) (*model.WavefrontFrame, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, run_id, seq, timestamp_ms, star_id, matrix_id, residuals, status, checksum, created_at
		 FROM wavefront_frames WHERE run_id = ? AND seq = ?`, runID, seq)
	return scanFrame(row)
}

// ListFramesByRun 列出某运行下的帧，按 seq 升序。
func (s *Store) ListFramesByRun(ctx context.Context, runID string) ([]*model.WavefrontFrame, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, seq, timestamp_ms, star_id, matrix_id, residuals, status, checksum, created_at
		 FROM wavefront_frames WHERE run_id = ? ORDER BY seq ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.WavefrontFrame
	for rows.Next() {
		var fr frameRow
		if err := rows.Scan(&fr.id, &fr.runID, &fr.seq, &fr.timestampMS, &fr.starID, &fr.matrixID,
			&fr.residuals, &fr.status, &fr.checksum, &fr.createdAt); err != nil {
			return nil, err
		}
		f, err := frameFromRow(fr)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UpdateFrameStatus 更新帧状态（如排除/恢复）。
func (s *Store) UpdateFrameStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE wavefront_frames SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return mapErr(err, "波前帧")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: 波前帧", model.ErrNotFound)
	}
	return nil
}

// CountAnalyzableFramesByRun 统计某运行下未排除的帧数。
func (s *Store) CountAnalyzableFramesByRun(ctx context.Context, runID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM wavefront_frames WHERE run_id = ? AND status != ?`,
		runID, model.FrameStatusExcluded).Scan(&n)
	return n, err
}

// CountFramesByRun 统计某运行下的帧数。
func (s *Store) CountFramesByRun(ctx context.Context, runID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wavefront_frames WHERE run_id = ?`, runID).Scan(&n)
	return n, err
}

// CountFrames 统计全部帧数。
func (s *Store) CountFrames(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wavefront_frames`).Scan(&n)
	return n, err
}

func scanFrame(row *sql.Row) (*model.WavefrontFrame, error) {
	var fr frameRow
	if err := row.Scan(&fr.id, &fr.runID, &fr.seq, &fr.timestampMS, &fr.starID, &fr.matrixID,
		&fr.residuals, &fr.status, &fr.checksum, &fr.createdAt); err != nil {
		return nil, mapErr(err, "波前帧")
	}
	return frameFromRow(fr)
}

func frameFromRow(fr frameRow) (*model.WavefrontFrame, error) {
	f := &model.WavefrontFrame{
		ID:          fr.id,
		RunID:       fr.runID,
		Seq:         fr.seq,
		TimestampMS: fr.timestampMS,
		StarID:      fr.starID,
		MatrixID:    fr.matrixID,
		Status:      fr.status,
		Checksum:    fr.checksum,
		CreatedAt:   parseTime(fr.createdAt),
	}
	if err := json.Unmarshal([]byte(fr.residuals), &f.Residuals); err != nil {
		return nil, fmt.Errorf("解析残差: %w", err)
	}
	return f, nil
}
