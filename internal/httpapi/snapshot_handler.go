package httpapi

import (
	"net/http"

	"task269-wavecalib/internal/model"
)

// handleCreateSnapshot 为运行创建诊断快照（草稿）。
func (h *Handler) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	var body struct {
		BaselineMatrixID string `json:"baseline_matrix_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	snap, err := h.svc.Snaps.CreateSnapshot(r.Context(), runID, body.BaselineMatrixID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

// handlePublishSnapshot 发布草稿快照。
func (h *Handler) handlePublishSnapshot(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	snap, err := h.svc.Snaps.PublishSnapshot(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// handleListSnapshots 列出运行的快照。
func (h *Handler) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	snaps, err := h.svc.Snaps.ListSnapshots(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if snaps == nil {
		snaps = []*model.DiagnosisSnapshot{}
	}
	writeJSON(w, http.StatusOK, snaps)
}

// handleGetSnapshot 获取单个快照。
func (h *Handler) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	snap, err := h.svc.Snaps.GetSnapshot(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}
