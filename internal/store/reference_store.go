package store

import (
	"context"
	"database/sql"
	"fmt"

	"task269-wavecalib/internal/model"
)

// CreateReferenceStar 登记参考星；name 唯一冲突返回 ErrConflict。
func (s *Store) CreateReferenceStar(ctx context.Context, st *model.ReferenceStar) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reference_stars (id, name, snr, position_error_arcsec, quality_score, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		st.ID, st.Name, st.SNR, st.PositionErrorArcsec, st.QualityScore, st.Status, now())
	if err != nil {
		return mapErr(err, "参考星")
	}
	st.CreatedAt = parseTime(now())
	return nil
}

// GetReferenceStar 按 ID 读取参考星。
func (s *Store) GetReferenceStar(ctx context.Context, id string) (*model.ReferenceStar, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, snr, position_error_arcsec, quality_score, status, created_at
		 FROM reference_stars WHERE id = ?`, id)
	var st model.ReferenceStar
	var created string
	if err := row.Scan(&st.ID, &st.Name, &st.SNR, &st.PositionErrorArcsec, &st.QualityScore, &st.Status, &created); err != nil {
		return nil, mapErr(err, "参考星")
	}
	st.CreatedAt = parseTime(created)
	return &st, nil
}

// ListReferenceStars 列出全部参考星。
func (s *Store) ListReferenceStars(ctx context.Context) ([]*model.ReferenceStar, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, snr, position_error_arcsec, quality_score, status, created_at
		 FROM reference_stars ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ReferenceStar
	for rows.Next() {
		var st model.ReferenceStar
		var created string
		if err := rows.Scan(&st.ID, &st.Name, &st.SNR, &st.PositionErrorArcsec, &st.QualityScore, &st.Status, &created); err != nil {
			return nil, err
		}
		st.CreatedAt = parseTime(created)
		out = append(out, &st)
	}
	return out, rows.Err()
}

// UpdateStarQuality 更新参考星质量分与状态。
func (s *Store) UpdateStarQuality(ctx context.Context, id string, score float64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE reference_stars SET quality_score = ?, status = ? WHERE id = ?`, score, status, id)
	if err != nil {
		return mapErr(err, "参考星")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: 参考星", model.ErrNotFound)
	}
	return nil
}

// CreateCalibrationMatrix 登记校准矩阵；version 唯一冲突返回 ErrConflict。
func (s *Store) CreateCalibrationMatrix(ctx context.Context, m *model.CalibrationMatrix) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO calibration_matrices (id, version, rows, cols, effective_from, effective_until, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Version, m.Rows, m.Cols, m.EffectiveFrom, m.EffectiveUntil, m.Status, now())
	if err != nil {
		return mapErr(err, "校准矩阵")
	}
	m.CreatedAt = parseTime(now())
	return nil
}

// GetCalibrationMatrix 按 ID 读取校准矩阵。
func (s *Store) GetCalibrationMatrix(ctx context.Context, id string) (*model.CalibrationMatrix, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, version, rows, cols, effective_from, effective_until, status, created_at
		 FROM calibration_matrices WHERE id = ?`, id)
	var m model.CalibrationMatrix
	var created string
	if err := row.Scan(&m.ID, &m.Version, &m.Rows, &m.Cols, &m.EffectiveFrom, &m.EffectiveUntil, &m.Status, &created); err != nil {
		return nil, mapErr(err, "校准矩阵")
	}
	m.CreatedAt = parseTime(created)
	return &m, nil
}

// ListCalibrationMatrices 列出全部校准矩阵。
func (s *Store) ListCalibrationMatrices(ctx context.Context) ([]*model.CalibrationMatrix, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, version, rows, cols, effective_from, effective_until, status, created_at
		 FROM calibration_matrices ORDER BY effective_from DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.CalibrationMatrix
	for rows.Next() {
		var m model.CalibrationMatrix
		var created string
		if err := rows.Scan(&m.ID, &m.Version, &m.Rows, &m.Cols, &m.EffectiveFrom, &m.EffectiveUntil, &m.Status, &created); err != nil {
			return nil, err
		}
		m.CreatedAt = parseTime(created)
		out = append(out, &m)
	}
	return out, rows.Err()
}

// UpdateMatrixStatus 更新矩阵状态（生效/过期/废止）。
func (s *Store) UpdateMatrixStatus(ctx context.Context, id, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE calibration_matrices SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return mapErr(err, "校准矩阵")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("%w: 校准矩阵", model.ErrNotFound)
	}
	return nil
}

// ActiveMatrixAt 返回在 ts 时刻生效的校准矩阵（effective_from <= ts < effective_until）。
func (s *Store) ActiveMatrixAt(ctx context.Context, ts int64) (*model.CalibrationMatrix, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, version, rows, cols, effective_from, effective_until, status, created_at
		 FROM calibration_matrices
		 WHERE effective_from <= ? AND effective_until > ? AND status = ?
		 ORDER BY effective_from DESC LIMIT 1`,
		ts, ts, model.MatrixStatusActive)
	var m model.CalibrationMatrix
	var created string
	if err := row.Scan(&m.ID, &m.Version, &m.Rows, &m.Cols, &m.EffectiveFrom, &m.EffectiveUntil, &m.Status, &created); err != nil {
		return nil, mapErr(err, "生效校准矩阵")
	}
	m.CreatedAt = parseTime(created)
	return &m, nil
}

var _ = sql.ErrNoRows
