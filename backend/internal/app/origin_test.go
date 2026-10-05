package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMutationOrigins(t *testing.T) {
	cases := []struct {
		name, env, appURL, origin, requestedWith string
		want                                     int
	}{
		{"configured", "development", "http://localhost:8080", "http://localhost:8080", "fsp", 404},
		{"IPv4 alias", "development", "http://localhost:8080", "http://127.0.0.1:8080", "fsp", 404},
		{"IPv6 alias", "development", "http://localhost:8080", "http://[::1]:8080", "fsp", 404},
		{"localhost alias", "development", "http://127.0.0.1:5173", "http://localhost:5173", "fsp", 404},
		{"CLI without origin", "production", "https://career.example", "", "fsp", 404},
		{"production configured", "production", "https://career.example", "https://career.example", "fsp", 404},
		{"missing application header", "development", "http://localhost:8080", "http://127.0.0.1:8080", "", 403},
		{"wrong application header", "development", "http://localhost:8080", "http://localhost:8080", "other", 403},
		{"different port", "development", "http://localhost:8080", "http://127.0.0.1:5173", "fsp", 403},
		{"different scheme", "development", "http://localhost:8080", "https://127.0.0.1:8080", "fsp", 403},
		{"external origin", "development", "http://localhost:8080", "https://attacker.example", "fsp", 403},
		{"localhost domain suffix", "development", "http://localhost:8080", "http://localhost.attacker.example:8080", "fsp", 403},
		{"IP domain suffix", "development", "http://localhost:8080", "http://127.0.0.1.attacker.example:8080", "fsp", 403},
		{"userinfo", "development", "http://localhost:8080", "http://attacker@127.0.0.1:8080", "fsp", 403},
		{"path", "development", "http://localhost:8080", "http://127.0.0.1:8080/login", "fsp", 403},
		{"query", "development", "http://localhost:8080", "http://127.0.0.1:8080?x=1", "fsp", 403},
		{"empty query", "development", "http://localhost:8080", "http://127.0.0.1:8080?", "fsp", 403},
		{"fragment", "development", "http://localhost:8080", "http://127.0.0.1:8080#", "fsp", 403},
		{"null origin", "development", "http://localhost:8080", "null", "fsp", 403},
		{"multiple origins", "development", "http://localhost:8080", "http://127.0.0.1:8080 http://attacker.example", "fsp", 403},
		{"malformed URL", "development", "http://localhost:8080", "http://[::1", "fsp", 403},
		{"production local alias", "production", "https://localhost:8080", "https://127.0.0.1:8080", "fsp", 403},
		{"unknown environment", "staging", "http://localhost:8080", "http://127.0.0.1:8080", "fsp", 403},
		{"external app URL", "development", "http://career.example:8080", "http://127.0.0.1:8080", "fsp", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{Config: Config{Env: tc.env, AppURL: tc.appURL}}
			// An unknown route requires no storage: 404 means the request passed
			// origin validation; forbidden mutations must stop at the middleware.
			h := a.Router()
			r := httptest.NewRequest(http.MethodPost, "/unknown", nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("X-Requested-With", tc.requestedWith)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
