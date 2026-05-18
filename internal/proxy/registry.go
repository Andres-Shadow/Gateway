package proxy

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"gateway/internal/models"
)

type RouteRegistry struct {
	mu     sync.RWMutex
	routes []CompiledRoute
}

type CompiledRoute struct {
	ID                    uint
	Path                  string
	TargetURL             string
	Methods               map[string]struct{}
	HealthCheckPath       string
	RewritePrefixFrom     string
	RewritePrefixTo       string
	RequestHeadersSet     map[string]string
	RequestHeadersRemove  []string
	ResponseHeadersSet    map[string]string
	ResponseHeadersRemove []string
	RequestBodyTransform  string
	ResponseBodyTransform string
	Healthy               bool
}

func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{}
}

func (r *RouteRegistry) Replace(routes []models.Route) {
	compiled := compileRoutes(routes)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = compiled
}

func (r *RouteRegistry) Upsert(route models.Route) {
	compiled := compileRoute(route)
	r.mu.Lock()
	defer r.mu.Unlock()

	replaced := false
	for i, existing := range r.routes {
		if existing.ID == compiled.ID {
			r.routes[i] = compiled
			replaced = true
			break
		}
	}
	if !replaced {
		r.routes = append(r.routes, compiled)
	}
	sortRoutes(r.routes)
}

func (r *RouteRegistry) Remove(id uint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, route := range r.routes {
		if route.ID == id {
			r.routes = append(r.routes[:i], r.routes[i+1:]...)
			return
		}
	}
}

func (r *RouteRegistry) Match(method, path string) (CompiledRoute, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, route := range r.routes {
		if route.Healthy && route.matchesMethod(method) && route.matchesPath(path) {
			return route, true
		}
	}
	return CompiledRoute{}, false
}

func (r *RouteRegistry) Snapshot() []CompiledRoute {
	r.mu.RLock()
	defer r.mu.RUnlock()
	routes := make([]CompiledRoute, len(r.routes))
	copy(routes, r.routes)
	return routes
}

func (r *RouteRegistry) SetHealthy(id uint, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.routes {
		if r.routes[i].ID == id {
			r.routes[i].Healthy = healthy
			return
		}
	}
}

func (r CompiledRoute) matchesMethod(method string) bool {
	_, ok := r.Methods[strings.ToUpper(method)]
	return ok
}

func (r CompiledRoute) matchesPath(path string) bool {
	if r.Path == path {
		return true
	}
	if !strings.HasSuffix(r.Path, "*") {
		return false
	}
	prefix := strings.TrimSuffix(r.Path, "*")
	base := strings.TrimRight(prefix, "/")
	if base == "" {
		return strings.HasPrefix(path, prefix)
	}
	return path == base || strings.HasPrefix(path, base+"/")
}

func compileRoutes(routes []models.Route) []CompiledRoute {
	compiled := make([]CompiledRoute, 0, len(routes))
	for _, route := range routes {
		compiled = append(compiled, compileRoute(route))
	}
	sortRoutes(compiled)
	return compiled
}

func compileRoute(route models.Route) CompiledRoute {
	methods := strings.Split(route.Methods, ",")
	compiledMethods := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method != "" {
			compiledMethods[method] = struct{}{}
		}
	}
	return CompiledRoute{
		ID:                    route.ID,
		Path:                  route.Path,
		TargetURL:             route.TargetURL,
		Methods:               compiledMethods,
		HealthCheckPath:       route.HealthCheckPath,
		RewritePrefixFrom:     route.RewritePrefixFrom,
		RewritePrefixTo:       route.RewritePrefixTo,
		RequestHeadersSet:     decodeStringMap(route.RequestHeadersSet),
		RequestHeadersRemove:  decodeStringSlice(route.RequestHeadersRemove),
		ResponseHeadersSet:    decodeStringMap(route.ResponseHeadersSet),
		ResponseHeadersRemove: decodeStringSlice(route.ResponseHeadersRemove),
		RequestBodyTransform:  route.RequestBodyTransform,
		ResponseBodyTransform: route.ResponseBodyTransform,
		Healthy:               true,
	}
}

func sortRoutes(routes []CompiledRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		return routeSpecificity(routes[i].Path) > routeSpecificity(routes[j].Path)
	})
}

func routeSpecificity(path string) int {
	return len(strings.TrimSuffix(path, "*"))
}

func decodeStringMap(value string) map[string]string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil
	}
	return decoded
}

func decodeStringSlice(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil
	}
	return decoded
}
