// Package httpapi 提供 HTTP 路由与处理器，路由统一以 /api 开头。
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"task269-wavecalib/internal/service"
)

// Handler 聚合 HTTP 处理器。
type Handler struct {
	svc    *service.Service
	logger *log.Logger
	mux    *http.ServeMux
}

// NewHandler 构造 HTTP 处理器并注册全部路由。
func NewHandler(svc *service.Service, logger *log.Logger) *Handler {
	h := &Handler{svc: svc, logger: logger, mux: http.NewServeMux()}
	h.routes()
	return h
}

// Handler 返回 http.Handler。
func (h *Handler) Handler() http.Handler {
	return h.mux
}

// routes 注册全部 API 路由。
func (h *Handler) routes() {
	// 运行
	h.mux.HandleFunc("POST /api/runs", h.handleCreateRun)
	h.mux.HandleFunc("GET /api/runs", h.handleListRuns)
	h.mux.HandleFunc("GET /api/runs/{id}", h.handleGetRun)
	h.mux.HandleFunc("PATCH /api/runs/{id}/status", h.handleTransitionRun)
	h.mux.HandleFunc("POST /api/runs/{id}/seal", h.handleSealRun)
	h.mux.HandleFunc("POST /api/runs/{id}/analyze", h.handleAnalyzeRun)

	// 波前帧
	h.mux.HandleFunc("POST /api/runs/{id}/frames", h.handleIngestFrame)
	h.mux.HandleFunc("GET /api/runs/{id}/frames", h.handleListFrames)
	h.mux.HandleFunc("GET /api/frames/{id}", h.handleGetFrame)
	h.mux.HandleFunc("POST /api/frames/{id}/exclude", h.handleExcludeFrame)
	h.mux.HandleFunc("POST /api/frames/{id}/restore", h.handleRestoreFrame)

	// 参考星
	h.mux.HandleFunc("POST /api/reference-stars", h.handleCreateStar)
	h.mux.HandleFunc("GET /api/reference-stars", h.handleListStars)
	h.mux.HandleFunc("GET /api/reference-stars/{id}", h.handleGetStar)
	h.mux.HandleFunc("POST /api/reference-stars/{id}/reevaluate", h.handleReevaluateStar)

	// 校准矩阵
	h.mux.HandleFunc("POST /api/calibration-matrices", h.handleCreateMatrix)
	h.mux.HandleFunc("GET /api/calibration-matrices", h.handleListMatrices)
	h.mux.HandleFunc("GET /api/calibration-matrices/{id}", h.handleGetMatrix)

	// 诊断
	h.mux.HandleFunc("POST /api/runs/{id}/residual-analysis", h.handleAnalyzeResiduals)
	h.mux.HandleFunc("GET /api/runs/{id}/residual-modes", h.handleListResidualModes)
	h.mux.HandleFunc("POST /api/runs/{id}/drift-candidates", h.handleAttributeDrift)
	h.mux.HandleFunc("GET /api/runs/{id}/drift-candidates", h.handleListCandidates)
	h.mux.HandleFunc("POST /api/candidates/{id}/confirm", h.handleConfirmCandidate)
	h.mux.HandleFunc("POST /api/candidates/{id}/reject", h.handleRejectCandidate)

	// 快照
	h.mux.HandleFunc("POST /api/runs/{id}/snapshots", h.handleCreateSnapshot)
	h.mux.HandleFunc("POST /api/snapshots/{id}/publish", h.handlePublishSnapshot)
	h.mux.HandleFunc("GET /api/runs/{id}/snapshots", h.handleListSnapshots)
	h.mux.HandleFunc("GET /api/snapshots/{id}", h.handleGetSnapshot)

	// 运维
	h.mux.HandleFunc("GET /api/health", h.handleHealth)
	h.mux.HandleFunc("GET /api/stats", h.handleStats)
}

// writeJSON 统一写 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr 统一写错误响应。
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// pathValue 读取路由参数。
func pathValue(r *http.Request, key string) string {
	return r.PathValue(key)
}
