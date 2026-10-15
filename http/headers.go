package fbhttp

import "net/http"

// defaultHeaders are applied to every response leaving the router. Handlers run
// after the middleware and may override individual values (the static handler
// sets its own long-lived Cache-Control).
var defaultHeaders = map[string]string{
	"Content-Security-Policy": "default-src 'self'; style-src 'unsafe-inline';",
	"Cache-Control":           "no-cache, no-store, must-revalidate",
}

func withDefaultHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range defaultHeaders {
			w.Header().Set(k, v)
		}
		next.ServeHTTP(w, r)
	})
}
