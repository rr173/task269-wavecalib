package httpapi

import (
	"context"
	"net/http"
	"time"
)

// handleHealth 健康检查。
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.svc.Store.Ping(ctx); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "数据库不可用: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleStats 统计概览。
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	runs, _ := h.svc.Store.CountRuns(ctx)
	frames, _ := h.svc.Store.CountFrames(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"runs":   runs,
		"frames": frames,
	})
}
