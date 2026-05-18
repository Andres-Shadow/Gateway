package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	registry     *RouteRegistry
	breaker      *CircuitBreaker
	logger       *slog.Logger
	maxRetries   int
	retryBackoff time.Duration
}

type HandlerOptions struct {
	MaxRetries   int
	RetryBackoff time.Duration
}

func NewHandler(registry *RouteRegistry, breaker *CircuitBreaker, logger *slog.Logger, opts HandlerOptions) *Handler {
	if opts.MaxRetries < 0 {
		opts.MaxRetries = 0
	}
	if opts.RetryBackoff <= 0 {
		opts.RetryBackoff = 100 * time.Millisecond
	}
	return &Handler{registry: registry, breaker: breaker, logger: logger, maxRetries: opts.MaxRetries, retryBackoff: opts.RetryBackoff}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, ok := h.registry.Match(r.Method, r.URL.Path)
	if !ok {
		writeProxyError(w, http.StatusNotFound, "route not found")
		return
	}
	if h.breaker != nil && !h.breaker.Allow(route.ID) {
		writeProxyError(w, http.StatusServiceUnavailable, "upstream circuit is open")
		return
	}

	target, err := url.Parse(route.TargetURL)
	if err != nil {
		h.logger.Error("invalid target url in route cache", "route_id", route.ID, "target_url", route.TargetURL, "error", err)
		writeProxyError(w, http.StatusBadGateway, "invalid upstream target")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	body = transformBody(body, route.RequestBodyTransform)

	var last *bufferedResponseWriter
	attempts := h.maxRetries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(h.retryBackoff * time.Duration(attempt))
		}
		req := r.Clone(r.Context())
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.URL.Path = rewritePath(r.URL.Path, route)
		req.URL.RawPath = ""

		recorder := newBufferedResponseWriter()
		h.proxyOnce(recorder, req, target, route)
		last = recorder
		if recorder.status < http.StatusBadGateway {
			if h.breaker != nil {
				h.breaker.RecordSuccess(route.ID)
			}
			copyBufferedResponse(w, recorder, route)
			return
		}
	}

	if h.breaker != nil {
		h.breaker.RecordFailure(route.ID)
	}
	if last != nil {
		copyBufferedResponse(w, last, route)
		return
	}
	writeProxyError(w, http.StatusBadGateway, "upstream request failed")
}

func (h *Handler) proxyOnce(w http.ResponseWriter, r *http.Request, target *url.URL, route CompiledRoute) {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Gateway-Route-ID", routeID(route.ID))
		for key, value := range route.RequestHeadersSet {
			req.Header.Set(key, value)
		}
		for _, key := range route.RequestHeadersRemove {
			req.Header.Del(key)
		}
	}
	proxy.ModifyResponse = func(res *http.Response) error {
		for key, value := range route.ResponseHeadersSet {
			res.Header.Set(key, value)
		}
		for _, key := range route.ResponseHeadersRemove {
			res.Header.Del(key)
		}
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		h.logger.Error("upstream request failed", "route_id", route.ID, "target_url", route.TargetURL, "error", proxyErr)
		writeProxyError(rw, http.StatusBadGateway, "upstream request failed")
	}
	h.logger.Debug("proxying request", "route_id", route.ID, "method", r.Method, "path", r.URL.Path, "target_url", route.TargetURL)
	proxy.ServeHTTP(w, r)
}

func writeProxyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func routeID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

func rewritePath(path string, route CompiledRoute) string {
	if route.RewritePrefixFrom == "" {
		return path
	}
	from := strings.TrimRight(route.RewritePrefixFrom, "/")
	to := strings.TrimRight(route.RewritePrefixTo, "/")
	if path == from {
		if to == "" {
			return "/"
		}
		return to
	}
	if strings.HasPrefix(path, from+"/") {
		suffix := strings.TrimPrefix(path, from)
		if to == "" {
			return suffix
		}
		return to + suffix
	}
	return path
}

func transformBody(body []byte, transform string) []byte {
	switch transform {
	case "uppercase":
		return []byte(strings.ToUpper(string(body)))
	case "lowercase":
		return []byte(strings.ToLower(string(body)))
	default:
		return body
	}
}

type bufferedResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func copyBufferedResponse(dst http.ResponseWriter, src *bufferedResponseWriter, route CompiledRoute) {
	for key, values := range src.header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	for key, value := range route.ResponseHeadersSet {
		dst.Header().Set(key, value)
	}
	for _, key := range route.ResponseHeadersRemove {
		dst.Header().Del(key)
	}
	body := transformBody(src.body.Bytes(), route.ResponseBodyTransform)
	dst.Header().Set("Content-Length", strconv.Itoa(len(body)))
	dst.WriteHeader(src.status)
	_, _ = dst.Write(body)
}
