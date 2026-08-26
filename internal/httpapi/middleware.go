package httpapi

import (
	"context"
	"net/http"
	"time"
)

// withTimeout 为请求注入超时上下文。
func withTimeout(r *http.Request, d time.Duration) (*http.Request, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), d)
	return r.WithContext(ctx), cancel
}

// requestID 中间件为每个请求附加 X-Request-Id 头。
type requestIDKey struct{}

// RequestID 返回当前请求的请求 ID（未设置时为空）。
func RequestID(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// logMiddleware 记录请求方法、路径与耗时。
func (h *Handler) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if h.logger != nil {
			h.logger.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		}
	})
}

// methodNotAllowed 兜底响应。
func methodNotAllowed(w http.ResponseWriter) {
	writeErr(w, http.StatusMethodNotAllowed, "方法不允许")
}
