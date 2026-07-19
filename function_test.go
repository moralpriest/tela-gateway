package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"
)

const testSCID = "b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2"

func TestAppFromHost(t *testing.T) {
	t.Setenv("TELA_HOST_SUFFIX", ".tela.cypher-punks.com")
	t.Setenv("HOST_MAP", "")
	t.Setenv("TELA_ALIASES", "derobeats="+testSCID)
	t.Setenv("RESERVED_APPS", "")

	tests := []struct {
		name        string
		host        string
		wantApp     string
		wantMatched bool
		wantSCID    bool
	}{
		{"known app", "derobeats.tela.cypher-punks.com", "derobeats", true, true},
		{"reserved status", "status.tela.cypher-punks.com", "status", true, false},
		{"unknown app", "nope.tela.cypher-punks.com", "nope", true, false},
		{"multi level", "a.b.tela.cypher-punks.com", "a.b", true, false},
		{"apex", "cypher-punks.com", "", false, false},
		{"cloudfront domain", "d5m36yv82gbu5.cloudfront.net", "", false, false},
		{"suffix itself", "tela.cypher-punks.com", "", false, false},
		{"empty", "", "", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, scid, matched := appFromHost(tc.host)
			if app != tc.wantApp || matched != tc.wantMatched {
				t.Fatalf("appFromHost(%q) = (%q, %q, %v), want app=%q matched=%v",
					tc.host, app, scid, matched, tc.wantApp, tc.wantMatched)
			}
			if (scid != "") != tc.wantSCID {
				t.Fatalf("appFromHost(%q) scid=%q, wantSCID=%v", tc.host, scid, tc.wantSCID)
			}
		})
	}
}

func TestIsReservedApp(t *testing.T) {
	t.Setenv("RESERVED_APPS", "www, gateway")
	for _, c := range []struct {
		app  string
		want bool
	}{
		{"status", true},  // default
		{"STATUS", true},  // case-insensitive
		{"www", true},     // env
		{"gateway", true}, // env (trimmed)
		{"derobeats", false},
		{"", false},
	} {
		if got := isReservedApp(c.app); got != c.want {
			t.Errorf("isReservedApp(%q) = %v, want %v", c.app, got, c.want)
		}
	}
}

func TestServeTELARouting(t *testing.T) {
	t.Setenv("TELA_HOST_SUFFIX", ".tela.cypher-punks.com")
	t.Setenv("HOST_MAP", "")
	t.Setenv("TELA_ALIASES", "")
	t.Setenv("RESERVED_APPS", "")

	// Reserved status subdomain, root path → landing page.
	t.Run("status landing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Host", "status.tela.cypher-punks.com")
		rec := httptest.NewRecorder()
		ServeTELA(rec, req)
		if rec.Code != 200 {
			t.Fatalf("status=%d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "<title>tela-gateway</title>") {
			t.Fatalf("expected landing page, got:\n%s", rec.Body.String())
		}
	})

	// Health always works, even on a reserved/app host.
	t.Run("health on app host", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Forwarded-Host", "status.tela.cypher-punks.com")
		rec := httptest.NewRecorder()
		ServeTELA(rec, req)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"service":"tela-gateway"`) {
			t.Fatalf("health failed: %d %s", rec.Code, rec.Body.String())
		}
	})

	// Unknown app subdomain → 404 with helpful body.
	t.Run("unknown app 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Host", "derobeatss.tela.cypher-punks.com")
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		ServeTELA(rec, req)
		if rec.Code != 404 {
			t.Fatalf("status=%d, want 404", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "no such TELA app") || !strings.Contains(body, "derobeatss") {
			t.Fatalf("expected unknown-app page, got:\n%s", body)
		}
		if !strings.Contains(body, "https://status.tela.cypher-punks.com/") {
			t.Fatalf("expected status link, got:\n%s", body)
		}
	})

	// Multi-level host → 404.
	t.Run("multi level 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-Host", "a.b.tela.cypher-punks.com")
		rec := httptest.NewRecorder()
		ServeTELA(rec, req)
		if rec.Code != 404 {
			t.Fatalf("status=%d, want 404", rec.Code)
		}
	})

	// Bare CDN/apex host, root path → landing page.
	t.Run("cdn host landing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = "d5m36yv82gbu5.cloudfront.net"
		rec := httptest.NewRecorder()
		ServeTELA(rec, req)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>tela-gateway</title>") {
			t.Fatalf("expected landing, got %d:\n%s", rec.Code, rec.Body.String())
		}
	})
}
