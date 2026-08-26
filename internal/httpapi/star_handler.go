package httpapi

import (
	"net/http"

	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/service"
)

// handleCreateStar 登记参考星。
func (h *Handler) handleCreateStar(w http.ResponseWriter, r *http.Request) {
	var in service.StarInput
	if !decode(w, r, &in) {
		return
	}
	star, err := h.svc.Stars.RegisterStar(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, star)
}

// handleListStars 列出参考星。
func (h *Handler) handleListStars(w http.ResponseWriter, r *http.Request) {
	stars, err := h.svc.Stars.ListStars(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if stars == nil {
		stars = []*model.ReferenceStar{}
	}
	writeJSON(w, http.StatusOK, stars)
}

// handleGetStar 获取参考星。
func (h *Handler) handleGetStar(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	star, err := h.svc.Stars.GetStar(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, star)
}

// handleReevaluateStar 重新评估参考星质量。
func (h *Handler) handleReevaluateStar(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	star, err := h.svc.Stars.ReEvaluateStar(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, star)
}

// handleCreateMatrix 登记校准矩阵。
func (h *Handler) handleCreateMatrix(w http.ResponseWriter, r *http.Request) {
	var in service.MatrixInput
	if !decode(w, r, &in) {
		return
	}
	mat, err := h.svc.Matrices.RegisterMatrix(r.Context(), in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mat)
}

// handleListMatrices 列出校准矩阵。
func (h *Handler) handleListMatrices(w http.ResponseWriter, r *http.Request) {
	mats, err := h.svc.Matrices.ListMatrices(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if mats == nil {
		mats = []*model.CalibrationMatrix{}
	}
	writeJSON(w, http.StatusOK, mats)
}

// handleGetMatrix 获取校准矩阵。
func (h *Handler) handleGetMatrix(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	mat, err := h.svc.Matrices.GetMatrix(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mat)
}
