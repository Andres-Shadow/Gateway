package proxy

import "testing"

func TestWildcardMatchingRespectsPathSegments(t *testing.T) {
	route := CompiledRoute{Path: "/api/*"}

	tests := map[string]bool{
		"/api":       true,
		"/api/":      true,
		"/api/users": true,
		"/apix":      false,
		"/api-v1":    false,
	}

	for path, expected := range tests {
		if got := route.matchesPath(path); got != expected {
			t.Fatalf("matchesPath(%q) = %v, want %v", path, got, expected)
		}
	}
}
