package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"gateway/internal/models"
	"gateway/internal/proxy"
	"gateway/internal/repositories"
)

var ErrInvalidRoute = errors.New("invalid route")

type CreateRouteInput struct {
	Path                  string            `json:"path"`
	TargetURL             string            `json:"target_url"`
	Methods               string            `json:"methods"`
	IsActive              *bool             `json:"is_active,omitempty"`
	HealthCheckPath       string            `json:"health_check_path,omitempty"`
	RewritePrefixFrom     string            `json:"rewrite_prefix_from,omitempty"`
	RewritePrefixTo       string            `json:"rewrite_prefix_to,omitempty"`
	RequestHeadersSet     map[string]string `json:"request_headers_set,omitempty"`
	RequestHeadersRemove  []string          `json:"request_headers_remove,omitempty"`
	ResponseHeadersSet    map[string]string `json:"response_headers_set,omitempty"`
	ResponseHeadersRemove []string          `json:"response_headers_remove,omitempty"`
	RequestBodyTransform  string            `json:"request_body_transform,omitempty"`
	ResponseBodyTransform string            `json:"response_body_transform,omitempty"`
}

type UpdateRouteInput struct {
	Path                  *string            `json:"path,omitempty"`
	TargetURL             *string            `json:"target_url,omitempty"`
	Methods               *string            `json:"methods,omitempty"`
	IsActive              *bool              `json:"is_active,omitempty"`
	HealthCheckPath       *string            `json:"health_check_path,omitempty"`
	RewritePrefixFrom     *string            `json:"rewrite_prefix_from,omitempty"`
	RewritePrefixTo       *string            `json:"rewrite_prefix_to,omitempty"`
	RequestHeadersSet     *map[string]string `json:"request_headers_set,omitempty"`
	RequestHeadersRemove  *[]string          `json:"request_headers_remove,omitempty"`
	ResponseHeadersSet    *map[string]string `json:"response_headers_set,omitempty"`
	ResponseHeadersRemove *[]string          `json:"response_headers_remove,omitempty"`
	RequestBodyTransform  *string            `json:"request_body_transform,omitempty"`
	ResponseBodyTransform *string            `json:"response_body_transform,omitempty"`
}

type RouteService struct {
	repo     repositories.RouteRepository
	registry *proxy.RouteRegistry
}

func NewRouteService(repo repositories.RouteRepository, registry *proxy.RouteRegistry) *RouteService {
	return &RouteService{repo: repo, registry: registry}
}

func (s *RouteService) LoadActiveRoutes(ctx context.Context) error {
	routes, err := s.repo.ListActive(ctx)
	if err != nil {
		return err
	}
	s.registry.Replace(routes)
	return nil
}

func (s *RouteService) Create(ctx context.Context, input CreateRouteInput) (*models.Route, error) {
	route, err := buildRoute(input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, route); err != nil {
		return nil, err
	}
	s.syncRoute(*route)
	return route, nil
}

func (s *RouteService) List(ctx context.Context) ([]models.Route, error) {
	return s.repo.List(ctx)
}

func (s *RouteService) Get(ctx context.Context, id uint) (*models.Route, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RouteService) Update(ctx context.Context, id uint, input UpdateRouteInput) (*models.Route, error) {
	route, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Path != nil {
		route.Path = normalizePath(*input.Path)
	}
	if input.TargetURL != nil {
		route.TargetURL = strings.TrimSpace(*input.TargetURL)
	}
	if input.Methods != nil {
		route.Methods = normalizeMethods(*input.Methods)
	}
	if input.IsActive != nil {
		route.IsActive = *input.IsActive
	}
	if input.HealthCheckPath != nil {
		route.HealthCheckPath = normalizeOptionalPath(*input.HealthCheckPath)
	}
	if input.RewritePrefixFrom != nil {
		route.RewritePrefixFrom = normalizeOptionalPath(*input.RewritePrefixFrom)
	}
	if input.RewritePrefixTo != nil {
		route.RewritePrefixTo = normalizeOptionalPath(*input.RewritePrefixTo)
	}
	if input.RequestHeadersSet != nil {
		route.RequestHeadersSet = mustJSON(*input.RequestHeadersSet)
	}
	if input.RequestHeadersRemove != nil {
		route.RequestHeadersRemove = mustJSON(*input.RequestHeadersRemove)
	}
	if input.ResponseHeadersSet != nil {
		route.ResponseHeadersSet = mustJSON(*input.ResponseHeadersSet)
	}
	if input.ResponseHeadersRemove != nil {
		route.ResponseHeadersRemove = mustJSON(*input.ResponseHeadersRemove)
	}
	if input.RequestBodyTransform != nil {
		route.RequestBodyTransform = normalizeTransform(*input.RequestBodyTransform)
	}
	if input.ResponseBodyTransform != nil {
		route.ResponseBodyTransform = normalizeTransform(*input.ResponseBodyTransform)
	}
	if err := validateRoute(route.Path, route.TargetURL, route.Methods); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, route); err != nil {
		return nil, err
	}
	s.syncRoute(*route)
	return route, nil
}

func (s *RouteService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.registry.Remove(id)
	return nil
}

func (s *RouteService) syncRoute(route models.Route) {
	if route.IsActive {
		s.registry.Upsert(route)
		return
	}
	s.registry.Remove(route.ID)
}

func buildRoute(input CreateRouteInput) (*models.Route, error) {
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	route := &models.Route{
		Path:                  normalizePath(input.Path),
		TargetURL:             strings.TrimSpace(input.TargetURL),
		Methods:               normalizeMethods(input.Methods),
		IsActive:              active,
		HealthCheckPath:       normalizeOptionalPath(input.HealthCheckPath),
		RewritePrefixFrom:     normalizeOptionalPath(input.RewritePrefixFrom),
		RewritePrefixTo:       normalizeOptionalPath(input.RewritePrefixTo),
		RequestHeadersSet:     mustJSON(input.RequestHeadersSet),
		RequestHeadersRemove:  mustJSON(input.RequestHeadersRemove),
		ResponseHeadersSet:    mustJSON(input.ResponseHeadersSet),
		ResponseHeadersRemove: mustJSON(input.ResponseHeadersRemove),
		RequestBodyTransform:  normalizeTransform(input.RequestBodyTransform),
		ResponseBodyTransform: normalizeTransform(input.ResponseBodyTransform),
	}
	if err := validateRoute(route.Path, route.TargetURL, route.Methods); err != nil {
		return nil, err
	}
	return route, nil
}

func validateRoute(path, targetURL, methods string) error {
	if path == "" || !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: path must start with /", ErrInvalidRoute)
	}
	if strings.Contains(path, "**") {
		return fmt.Errorf("%w: path contains an invalid wildcard", ErrInvalidRoute)
	}
	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("%w: target_url must be an absolute URL", ErrInvalidRoute)
	}
	if methods == "" {
		return fmt.Errorf("%w: methods cannot be empty", ErrInvalidRoute)
	}
	return nil
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func normalizeOptionalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return normalizePath(path)
}

func normalizeMethods(methods string) string {
	parts := strings.Split(methods, ",")
	seen := make(map[string]struct{}, len(parts))
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		method := strings.ToUpper(strings.TrimSpace(part))
		if method == "" {
			continue
		}
		if _, exists := seen[method]; exists {
			continue
		}
		seen[method] = struct{}{}
		normalized = append(normalized, method)
	}
	sort.Strings(normalized)
	return strings.Join(normalized, ",")
}

func normalizeTransform(transform string) string {
	transform = strings.ToLower(strings.TrimSpace(transform))
	switch transform {
	case "", "none":
		return ""
	case "uppercase", "lowercase":
		return transform
	default:
		return transform
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil || string(data) == "null" {
		return ""
	}
	return string(data)
}

func ParseID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid id", ErrInvalidRoute)
	}
	return uint(id), nil
}
