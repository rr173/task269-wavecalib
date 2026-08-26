package httpapi

import (
	"net/http"

	"task269-wavecalib/internal/model"
)

// handleAnalyzeResiduals 对运行执行残差模式分析。
func (h *Handler) handleAnalyzeResiduals(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	result, err := h.svc.Diagnoses.AnalyzeRunResiduals(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleListResidualModes 列出运行的残差模式。
func (h *Handler) handleListResidualModes(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	modes, err := h.svc.Store.ListResidualModesByRun(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if modes == nil {
		modes = []*model.ResidualMode{}
	}
	writeJSON(w, http.StatusOK, modes)
}

// handleAttributeDrift 对运行执行漂移归因。
func (h *Handler) handleAttributeDrift(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	cands, err := h.svc.Diagnoses.AttributeDrift(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if cands == nil {
		cands = []*model.DriftCandidate{}
	}
	writeJSON(w, http.StatusCreated, cands)
}

// handleListCandidates 列出运行的漂移候选。
func (h *Handler) handleListCandidates(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	cands, err := h.svc.Diagnoses.ListCandidates(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if cands == nil {
		cands = []*model.DriftCandidate{}
	}
	writeJSON(w, http.StatusOK, cands)
}

// handleConfirmCandidate 确认漂移候选。
func (h *Handler) handleConfirmCandidate(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	cand, err := h.svc.Diagnoses.ConfirmCandidate(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cand)
}

// handleRejectCandidate 否决漂移候选。
func (h *Handler) handleRejectCandidate(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	cand, err := h.svc.Diagnoses.RejectCandidate(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cand)
}
