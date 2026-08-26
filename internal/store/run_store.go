package store

import (
	"context"
	"database/sql"
	"fmt"

	"task269-wavecalib/internal/model"
)

// CreateRun 插入一条观测运行；ID 冲突时返回 ErrConflict。
func (s *Store) CreateRun(ctx context.Context, r *model.ObservationRun) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO runs (id, name, telescope_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.TelescopeID, r.Status, now(), now())
	if err != nil {
		return mapErr(err, "运行")
	}
	r.CreatedAt = parseTime(now())
	r.UpdatedAt = r.CreatedAt
	return nil
}

// GetRun 按 ID 读取运行。
func (s *Store) GetRun(ctx context.Context, id string) (*model.ObservationRun, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, telescope_id, status, created_at, updated_at FROM runs WHERE id = ?`, id)
	var r model.ObservationRun
	var created, updated string
	if err := row.Scan(&r.ID, &r.Name, &r.TelescopeID, &r.Status, &created, &updated); err != nil {
		return nil, mapErr(err, "运行")
	}
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)
	return &r, nil
}

// ListRuns 按状态过滤列出运行；status 为空表示全部。
func (s *Store) ListRuns(ctx context.Context, status string) ([]*model.ObservationRun, error) {
	q := `SELECT id, name, telescope_id, status, created_at, updated_at FROM runs`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ObservationRun
	for rows.Next() {
		var r model.ObservationRun
		var created, updated string
		if err := rows.Scan(&r.ID, &r.Name, &r.TelescopeID, &r.Status, &created, &updated); err != nil {
			return nil, err
		}
		r.CreatedAt = parseTime(created)
		r.UpdatedAt = parseTime(updated)
		out = append(out, &r)
	}
	return out, rows.Err()
}

// UpdateRunStatus 用 CAS 语义更新运行状态：仅当当前状态等于 expected 时更新，
// 防止并发把状态机推进到非法路径。
func (s *Store) UpdateRunStatus(ctx context.Context, id, expected, next string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE runs SET status = ?, updated_at = ? WHERE id = ?`,
		next, now(), id)
	if err != nil {
		return mapErr(err, "运行")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var cur string
		if err := s.db.QueryRowContext(ctx, `SELECT status FROM runs WHERE id = ?`, id).Scan(&cur); err != nil {
			return mapErr(err, "运行")
		}
		if cur == next {
			return nil // 已是目标状态，视为成功
		}
		return fmt.Errorf("%w: 运行 %s 当前状态 %s，期望 %s", model.ErrInvalidState, id, cur, expected)
	}
	return nil
}

// CountRuns 统计运行数量（供健康统计）。
func (s *Store) CountRuns(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs`).Scan(&n)
	return n, err
}

var _ = sql.ErrNoRows // 保持 sql 导入引用（mapErr 使用）
