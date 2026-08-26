package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task269-wavecalib/internal/model"
)

// decode 解析 JSON 请求体到目标结构。
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体 JSON 解析失败: "+err.Error())
		return false
	}
	return true
}

// writeDomainError 把领域错误映射为 HTTP 状态码。
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.Is(err, model.ErrConflict):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrInvalidState):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrSealed):
		writeErr(w, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrInvalidInput):
		writeErr(w, http.StatusBadRequest, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "服务器内部错误: "+err.Error())
	}
}
