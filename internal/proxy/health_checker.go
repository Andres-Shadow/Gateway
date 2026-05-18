package proxy

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type HealthChecker struct {
	registry *RouteRegistry
	client   *http.Client
	interval time.Duration
	logger   *slog.Logger
}

func NewHealthChecker(registry *RouteRegistry, timeout, interval time.Duration, logger *slog.Logger) *HealthChecker {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &HealthChecker{
		registry: registry,
		client:   &http.Client{Timeout: timeout},
		interval: interval,
		logger:   logger,
	}
}

func (h *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	h.check(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.check(ctx)
		}
	}
}

func (h *HealthChecker) check(ctx context.Context) {
	for _, route := range h.registry.Snapshot() {
		healthy := h.routeHealthy(ctx, route)
		h.registry.SetHealthy(route.ID, healthy)
		if !healthy {
			h.logger.Warn("upstream health check failed", "route_id", route.ID, "target_url", route.TargetURL)
		}
	}
}

func (h *HealthChecker) routeHealthy(ctx context.Context, route CompiledRoute) bool {
	target, err := url.Parse(route.TargetURL)
	if err != nil {
		return false
	}
	checkPath := route.HealthCheckPath
	if checkPath == "" {
		checkPath = "/healthz"
	}
	target.Path = checkPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false
	}
	res, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode >= 200 && res.StatusCode < 500
}
