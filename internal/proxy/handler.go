package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
)

type Handler struct {
	registry *RouteRegistry
	logger   *slog.Logger
}

func NewHandler(registry *RouteRegistry, logger *slog.Logger) *Handler {
	return &Handler{registry: registry, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, ok := h.registry.Match(r.Method, r.URL.Path)
	if !ok {
		writeProxyError(w, http.StatusNotFound, "route not found")
		return
	}

	target, err := url.Parse(route.TargetURL)
	if err != nil {
		h.logger.Error("invalid target url in route cache", "route_id", route.ID, "target_url", route.TargetURL, "error", err)
		writeProxyError(w, http.StatusBadGateway, "invalid upstream target")
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Gateway-Route-ID", routeID(route.ID))
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
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

func routeID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
