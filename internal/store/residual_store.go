package store

import (
	"context"
	"database/sql"
	"fmt"

	"task269-wavecalib/internal/model"
)

// DeleteResidualModesByRun 删除某运行的全部残差模式，供重复分析前清理。
func (s *Store) DeleteResidualModesByRun(ctx context.Context, runID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM residual_modes WHERE run_id = ?`, runID)
	return err
}

// CreateResidualMode 写入一条残差模式分解结果；(frame_id, mode_name) 唯一。
func (s *Store) CreateResidualMode(ctx context.Context, rm *model.ResidualMode) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO residual_modes (id, run_id, frame_id, mode_name, coefficient, deviation, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rm.ID, rm.RunID, rm.FrameID, rm.ModeName, rm.Coefficient, rm.Deviation, now())
	if err != nil {
		return mapErr(err, "残差模式")
	}
	rm.CreatedAt = parseTime(now())
	return nil
}

// ListResidualModesByRun 列出某运行的全部残差模式。
func (s *Store) ListResidualModesByRun(ctx context.Context, runID string) ([]*model.ResidualMode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, frame_id, mode_name, coefficient, deviation, created_at
		 FROM residual_modes WHERE run_id = ? ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ResidualMode
	for rows.Next() {
		var rm model.ResidualMode
		var created string
		if err := rows.Scan(&rm.ID, &rm.RunID, &rm.FrameID, &rm.ModeName, &rm.Coefficient, &rm.Deviation, &created); err != nil {
			return nil, err
		}
		rm.CreatedAt = parseTime(created)
		out = append(out, &rm)
	}
	return out, rows.Err()
}

// CreateDriftCandidate 写入漂移候选。
func (s *Store) CreateDriftCandidate(ctx context.Context, c *model.DriftCandidate) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO drift_candidates (id, run_id, frame_id, matrix_id, attribution, confidence, detail, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.RunID, c.FrameID, c.MatrixID, c.Attribution, c.Confidence, c.Detail, c.Status, now())
	if err != nil {
		return mapErr(err, "漂移候选")
	}
	c.CreatedAt = parseTime(now())
	return nil
}

// GetDriftCandidate 按 ID 读取漂移候选。
func (s *Store) GetDriftCandidate(ctx context.Context, id string) (*model.DriftCandidate, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, run_id, frame_id, matrix_id, attribution, confidence, detail, status, created_at
		 FROM drift_candidates WHERE id = ?`, id)
	var c model.DriftCandidate
	var created string
	if err := row.Scan(&c.ID, &c.RunID, &c.FrameID, &c.MatrixID, &c.Attribution, &c.Confidence, &c.Detail, &c.Status, &created); err != nil {
		return nil, mapErr(err, "漂移候选")
	}
	c.CreatedAt = parseTime(created)
	return &c, nil
}

// ListDriftCandidatesByRun 列出某运行的漂移候选，按置信度降序。
func (s *Store) ListDriftCandidatesByRun(ctx context.Context, runID string) ([]*model.DriftCandidate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, frame_id, matrix_id, attribution, confidence, detail, status, created_at
		 FROM drift_candidates WHERE run_id = ? ORDER BY confidence DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.DriftCandidate
	for rows.Next() {
		var c model.DriftCandidate
		var created string
		if err := rows.Scan(&c.ID, &c.RunID, &c.FrameID, &c.MatrixID, &c.Attribution, &c.Confidence, &c.Detail, &c.Status, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTime(created)
		out = append(out, &c)
	}
	return out, rows.Err()
}

// UpdateDriftCandidateStatus 更新候选状态（确认/否决）。
func (s *Store) UpdateDriftCandidateStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE drift_candidates SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return mapErr(err, "漂移候选")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: 漂移候选", model.ErrNotFound)
	}
	return nil
}

// CountCandidatesByRun 统计某运行的候选数量。
func (s *Store) CountCandidatesByRun(ctx context.Context, runID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM drift_candidates WHERE run_id = ?`, runID).Scan(&n)
	return n, err
}

var _ = sql.ErrNoRows
