// Package store 提供 SQLite 持久化：建表迁移、CRUD 与重启恢复。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"task269-wavecalib/internal/model"
)

// Store 是 SQLite 存储句柄，所有表操作经由它执行。
type Store struct {
	db *sql.DB
}

// Open 打开（或创建）数据库文件并执行建表迁移。
// dir 为空时使用默认目录；文件不存在会自动创建。
func Open(path string) (*Store, error) {
	if path == "" {
		path = "wavecalib.db"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录: %w", err)
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping 检查数据库连通性。
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// migrate 执行幂等建表迁移。
func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS runs (
	id            TEXT PRIMARY KEY,
	name          TEXT NOT NULL,
	telescope_id  TEXT NOT NULL,
	status        TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reference_stars (
	id                   TEXT PRIMARY KEY,
	name                 TEXT NOT NULL UNIQUE,
	snr                  REAL NOT NULL,
	position_error_arcsec REAL NOT NULL,
	quality_score        REAL NOT NULL,
	status               TEXT NOT NULL,
	created_at           TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS calibration_matrices (
	id             TEXT PRIMARY KEY,
	version        TEXT NOT NULL UNIQUE,
	rows           INTEGER NOT NULL,
	cols           INTEGER NOT NULL,
	effective_from INTEGER NOT NULL,
	effective_until INTEGER NOT NULL,
	status         TEXT NOT NULL,
	created_at     TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS wavefront_frames (
	id          TEXT PRIMARY KEY,
	run_id      TEXT NOT NULL REFERENCES runs(id),
	seq         INTEGER NOT NULL,
	timestamp_ms INTEGER NOT NULL,
	star_id     TEXT NOT NULL REFERENCES reference_stars(id),
	matrix_id   TEXT NOT NULL REFERENCES calibration_matrices(id),
	residuals   TEXT NOT NULL,
	status      TEXT NOT NULL,
	checksum    TEXT NOT NULL,
	created_at  TEXT NOT NULL,
	UNIQUE(run_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_frames_run ON wavefront_frames(run_id, seq);
CREATE TABLE IF NOT EXISTS residual_modes (
	id          TEXT PRIMARY KEY,
	run_id      TEXT NOT NULL,
	frame_id    TEXT NOT NULL REFERENCES wavefront_frames(id),
	mode_name   TEXT NOT NULL,
	coefficient REAL NOT NULL,
	deviation   REAL NOT NULL,
	created_at  TEXT NOT NULL,
	UNIQUE(frame_id, mode_name)
);
CREATE TABLE IF NOT EXISTS drift_candidates (
	id          TEXT PRIMARY KEY,
	run_id      TEXT NOT NULL,
	frame_id    TEXT NOT NULL REFERENCES wavefront_frames(id),
	matrix_id   TEXT NOT NULL REFERENCES calibration_matrices(id),
	attribution TEXT NOT NULL,
	confidence  REAL NOT NULL,
	detail      TEXT NOT NULL,
	status      TEXT NOT NULL,
	created_at  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS diagnosis_snapshots (
	id                TEXT PRIMARY KEY,
	run_id            TEXT NOT NULL REFERENCES runs(id),
	version           INTEGER NOT NULL,
	baseline_matrix_id TEXT NOT NULL,
	content           TEXT NOT NULL,
	status            TEXT NOT NULL,
	created_at        TEXT NOT NULL,
	UNIQUE(run_id, version)
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("执行迁移: %w", err)
	}
	return nil
}

// now 返回统一格式的时间戳字符串。
func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// parseTime 解析存储的时间字符串；失败时回退为零值。
func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, v)
	return t
}

// 错误映射辅助：把 sqlite 约束错误转为领域错误。
func mapErr(err error, notFoundMsg string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", model.ErrNotFound, notFoundMsg)
	}
	if isConstraintErr(err) {
		return fmt.Errorf("%w: %v", model.ErrConflict, err)
	}
	return err
}

// isConstraintErr 识别 SQLite 唯一/外键约束错误。
func isConstraintErr(err error) bool {
	msg := err.Error()
	return len(msg) >= 19 && (msg[:19] == "constraint failed:" || msg[:19] == "constraint failed ")
}
