package httpapi

import (
	"net/http"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// handleIngestFrame 接收一帧波前（幂等）。
func (h *Handler) handleIngestFrame(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	var in acquisition.FrameInput
	if !decode(w, r, &in) {
		return
	}
	in.RunID = runID
	frame, err := h.svc.Frames.IngestFrame(r.Context(), &in)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, frame)
}

// handleListFrames 列出某运行的帧。
func (h *Handler) handleListFrames(w http.ResponseWriter, r *http.Request) {
	runID := pathValue(r, "id")
	frames, err := h.svc.Frames.ListFrames(r.Context(), runID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if frames == nil {
		frames = []*model.WavefrontFrame{}
	}
	writeJSON(w, http.StatusOK, frames)
}

// handleGetFrame 获取单个帧。
func (h *Handler) handleGetFrame(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	frame, err := h.svc.Frames.GetFrame(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frame)
}

// handleExcludeFrame 排除失真帧。
func (h *Handler) handleExcludeFrame(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	frame, err := h.svc.Frames.ExcludeFrame(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frame)
}

// handleRestoreFrame 恢复被排除的帧。
func (h *Handler) handleRestoreFrame(w http.ResponseWriter, r *http.Request) {
	id := pathValue(r, "id")
	frame, err := h.svc.Frames.RestoreFrame(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frame)
}
