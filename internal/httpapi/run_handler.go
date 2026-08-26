package httpapi

import (
	"net/http"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/service"
)

// handleCreateRun 创建观测运行。
func (h *Handler) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var in service.RunInput
	if !decode(w, r, &in) {
		return
	}
	run, err := h.svc.Runs.CreateRun(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// handleListRuns 列出运行，可按 status 过滤。
func (h *Handler) handleListRuns(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	runs, err := h.svc.Runs.ListRuns(r.Context(), status)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if runs == nil {
		runs = []*model.ObservationRun{}
	}
	writeJSON(w, http.StatusOK, runs)
}

// handleGetRun 获取单个运行。
func (h *Handler) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	run, err := h.svc.Runs.GetRun(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleTransitionRun 更新运行状态。
func (h *Handler) handleTransitionRun(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	var body struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &body) {
		return
	}
	run, err := h.svc.Runs.TransitionRun(r.Context(), id, body.Status)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleSealRun 封存运行。
func (h *Handler) handleSealRun(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	run, err := h.svc.Runs.SealRun(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleAnalyzeRun 把运行推进到待分析状态。
func (h *Handler) handleAnalyzeRun(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	run, err := h.svc.Runs.AnalyzeRun(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}
