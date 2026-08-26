package store

import (
	"context"
	"database/sql"
	"fmt"

	"task269-wavecalib/internal/model"
)

// CreateSnapshot 插入诊断快照；(run_id, version) 唯一。
func (s *Store) CreateSnapshot(ctx context.Context, snap *model.DiagnosisSnapshot) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO diagnosis_snapshots (id, run_id, version, baseline_matrix_id, content, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		snap.ID, snap.RunID, snap.Version, snap.BaselineMatrixID, snap.Content, snap.Status, now())
	if err != nil {
		return mapErr(err, "诊断快照")
	}
	snap.CreatedAt = parseTime(now())
	return nil
}

// GetSnapshot 按 ID 读取诊断快照。
func (s *Store) GetSnapshot(ctx context.Context, id string) (*model.DiagnosisSnapshot, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, run_id, version, baseline_matrix_id, content, status, created_at
		 FROM diagnosis_snapshots WHERE id = ?`, id)
	var snap model.DiagnosisSnapshot
	var created string
	if err := row.Scan(&snap.ID, &snap.RunID, &snap.Version, &snap.BaselineMatrixID, &snap.Content, &snap.Status, &created); err != nil {
		return nil, mapErr(err, "诊断快照")
	}
	snap.CreatedAt = parseTime(created)
	return &snap, nil
}

// ListSnapshotsByRun 列出某运行的快照，按版本降序。
func (s *Store) ListSnapshotsByRun(ctx context.Context, runID string) ([]*model.DiagnosisSnapshot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, version, baseline_matrix_id, content, status, created_at
		 FROM diagnosis_snapshots WHERE run_id = ? ORDER BY version DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.DiagnosisSnapshot
	for rows.Next() {
		var snap model.DiagnosisSnapshot
		var created string
		if err := rows.Scan(&snap.ID, &snap.RunID, &snap.Version, &snap.BaselineMatrixID, &snap.Content, &snap.Status, &created); err != nil {
			return nil, err
		}
		snap.CreatedAt = parseTime(created)
		out = append(out, &snap)
	}
	return out, rows.Err()
}

// LatestSnapshotVersion 返回某运行当前最大快照版本；无快照时返回 0。
func (s *Store) LatestSnapshotVersion(ctx context.Context, runID string) (int, error) {
	var v sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(version) FROM diagnosis_snapshots WHERE run_id = ?`, runID).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// UpdateSnapshotStatus 更新快照状态（发布/替代）。
func (s *Store) UpdateSnapshotStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE diagnosis_snapshots SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return mapErr(err, "诊断快照")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: 诊断快照", model.ErrNotFound)
	}
	return nil
}
