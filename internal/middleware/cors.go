package middleware

import (
	"net/http"
	"strings"
)

type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func CORS(opts CORSOptions) func(http.Handler) http.Handler {
	origins := make(map[string]struct{}, len(opts.AllowedOrigins))
	allowAll := false
	for _, origin := range opts.AllowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		origins[origin] = struct{}{}
	}
	methods := strings.Join(defaultIfEmpty(opts.AllowedMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}), ", ")
	headers := strings.Join(defaultIfEmpty(opts.AllowedHeaders, []string{"Authorization", "Content-Type"}), ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := origins[origin]; allowAll || ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Methods", methods)
					w.Header().Set("Access-Control-Allow-Headers", headers)
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func defaultIfEmpty(values, fallback []string) []string {
	if len(values) == 0 {
		return fallback
	}
	return values
}
