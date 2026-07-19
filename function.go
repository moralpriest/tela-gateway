package gateway

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/civilware/tela"
)

// Known dURL → INDEX SCID (exact match, case-insensitive).
// Not all apps use ".tela"; /scid/{hex}/ works for any INDEX without this map.
var aliases = map[string]string{
	"derobeats.tela": "b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2",
	"derobeats":      "b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2",
	"explorer.tela":  "9101c5d2d84adf0566f7980254d71d1d16d8d61e01786820233b8d4b7b2626d0",
	"explorer":       "9101c5d2d84adf0566f7980254d71d1d16d8d61e01786820233b8d4b7b2626d0",
}

// builtinAliases are always present as a last-resort fallback. The indexer's
// generated list (aliases.json / S3) and TELA_ALIASES env extend these.
var builtinAliases = map[string]string{
	"derobeats": "b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2",
	"explorer":  "9101c5d2d84adf0566f7980254d71d1d16d8d61e01786820233b8b2626d0",
}

var (
	once  sync.Once
	mu    sync.Mutex
	roots = map[string]string{}
)

func dataDir() string {
	if v := os.Getenv("TELA_DATA_DIR"); v != "" {
		return v
	}
	// Lambda only allows writes under /tmp
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return "/tmp/tela-gateway"
	}
	return filepath.Join(os.TempDir(), "tela-gateway")
}

func initTELA() {
	once.Do(func() {
		base := dataDir()
		if err := os.MkdirAll(base, 0o755); err != nil {
			panic("tela-gateway data dir: " + err.Error())
		}
		if err := tela.SetShardPath(base); err != nil {
			// Parent of base must exist (shards.SetPath). Ensure /tmp then retry.
			_ = os.MkdirAll(os.TempDir(), 0o1777)
			fallback := filepath.Join(os.TempDir(), "tela-gateway")
			_ = os.MkdirAll(fallback, 0o755)
			if err2 := tela.SetShardPath(fallback); err2 != nil {
				panic("tela-gateway SetShardPath: " + err.Error() + "; fallback: " + err2.Error())
			}
		}
		tela.AllowUpdates(true)
	})
}

// daemon returns the best available daemon endpoint (with failover).
// Deprecated alias kept for any external callers; prefer pickDaemon().
func daemon() string {
	return pickDaemon()
}

// ServeTELA is the HTTP entrypoint (local, VPS, Lambda Web Adapter, Cloud Functions).
func ServeTELA(w http.ResponseWriter, r *http.Request) {
	initTELA()

	path := strings.Trim(r.URL.Path, "/")

	// Optional vanity host: HOST_MAP=derobeats.example.com=b1e1…,other.com=abcd…
	// Behind a CDN the original viewer Host arrives in X-Forwarded-Host; prefer
	// it over r.Host (which is the CDN/origin domain). Local/direct access has
	// no X-Forwarded-Host, so r.Host is used as the fallback.
	host := r.Host
	if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
		host = strings.TrimSpace(strings.Split(fh, ",")[0])
	}
	app, scid, matched := appFromHost(host)
	if matched &&
		path != "health" &&
		!strings.HasPrefix(path, "scid/") &&
		!strings.HasPrefix(path, "durl/") {
		switch {
		case isReservedApp(app):
			// Reserved names (e.g. status) serve the gateway home / info
			// pages; fall through to the normal switch below.
		case scid != "":
			if path == "" {
				handleSCID(w, r, scid+"/")
			} else {
				handleSCID(w, r, scid+"/"+path)
			}
			return
		default:
			// Suffix matched but no such app (typo or multi-level host).
			writeUnknownApp(w, r, app)
			return
		}
	}

	switch {
	case path == "health":
		writeHealth(w)
		return
	case path == "":
		writeLanding(w, r)
		return
	case strings.HasPrefix(path, "durl/"):
		handleDURL(w, r, strings.TrimPrefix(path, "durl/"))
		return
	case strings.HasPrefix(path, "scid/"):
		handleSCID(w, r, strings.TrimPrefix(path, "scid/"))
		return
	default:
		http.Error(w, "use /scid/{64-hex-scid}/ or /durl/{name}", http.StatusNotFound)
	}
}

// appFromHost inspects the request host and reports whether it targets a
// per-app subdomain (via HOST_MAP or TELA_HOST_SUFFIX) rather than the bare
// gateway/CDN host. When matched is true the caller routes by app:
//
//   - scid != ""            → serve that TELA app
//   - isReservedApp(app)    → serve the gateway home (e.g. status.<suffix>)
//   - otherwise             → 404 (unknown app / multi-level host)
//
// When matched is false the host is the apex, CloudFront domain, or Lambda URL
// and the normal path-based switch applies.
func appFromHost(host string) (app string, scid string, matched bool) {
	host = strings.ToLower(strings.Split(host, ":")[0])
	if host == "" {
		return "", "", false
	}

	// 1. Explicit static host map: HOST_MAP=app.example.com=<scid>,...
	for _, part := range strings.Split(os.Getenv("HOST_MAP"), ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.ToLower(strings.TrimSpace(kv[0])) == host {
			s := strings.ToLower(strings.TrimSpace(kv[1]))
			if isSCID(s) {
				return host, s, true
			}
		}
	}

	// 2. Wildcard subdomain: <app>.cypher-punks.com → alias lookup for <app>.
	if suffix := os.Getenv("TELA_HOST_SUFFIX"); suffix != "" {
		suffix = strings.ToLower(strings.TrimPrefix(suffix, "."))
		if strings.HasSuffix(host, "."+suffix) && host != suffix {
			app = strings.TrimSuffix(host, "."+suffix)
			if app == "" {
				return "", "", false
			}
			// Multi-level hosts (foo.bar.<suffix>) match the suffix but are
			// not valid single-label apps: matched, no scid → 404.
			if strings.Contains(app, ".") {
				return app, "", true
			}
			if isReservedApp(app) {
				return app, "", true
			}
			return app, lookupAlias(app), true
		}
	}

	return "", "", false
}

// reservedApps are subdomain labels that always serve the gateway home/info
// pages instead of being resolved as on-chain TELA apps. Defaults to "status";
// extend via RESERVED_APPS=status,www,... (merged with the default).
var reservedApps = map[string]bool{"status": true}

func isReservedApp(app string) bool {
	app = strings.ToLower(strings.TrimSpace(app))
	if reservedApps[app] {
		return true
	}
	for _, part := range strings.Split(os.Getenv("RESERVED_APPS"), ",") {
		if strings.ToLower(strings.TrimSpace(part)) == app && app != "" {
			return true
		}
	}
	return false
}

// lookupAlias resolves a TELA app name (dURL or short alias) to its SCID,
// checking env overrides, the indexer's generated list, then built-ins.
func lookupAlias(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	// TELA_ALIASES env override (highest priority).
	for _, part := range strings.Split(os.Getenv("TELA_ALIASES"), ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.ToLower(strings.TrimSpace(kv[0])) == name {
			scid := strings.ToLower(strings.TrimSpace(kv[1]))
			if isSCID(scid) {
				return scid
			}
		}
	}
	// Generated indexer list (S3 / bundled aliases.json).
	if scid, ok := loadAliases()[name]; ok && isSCID(scid) {
		return scid
	}
	// Built-in fallback.
	if scid, ok := builtinAliases[name]; ok {
		return scid
	}
	return ""
}

func isSCID(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func writeHealth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"service": "tela-gateway",
		"daemon":  daemon(),
		"daemons": daemonListString(),
		"aliases": len(aliases),
		"note":    "any TELA INDEX via /scid/{scid}/; known names via /durl/{name}",
	})
}

func writeLanding(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(publicBase(r), "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>tela-gateway</title>
<style>
body{font-family:system-ui,sans-serif;max-width:42rem;margin:2rem auto;padding:0 1rem;line-height:1.5}
code,a{word-break:break-all} code{background:#f4f4f4;padding:.1em .3em}
</style></head><body>
<h1>tela-gateway</h1>
<p>Public TELA gateway — serves on-chain sites from a DERO node. No browser extension required to <em>view</em>.</p>
<ul>
<li><a href="%[1]s/health"><code>/health</code></a></li>
<li><a href="%[1]s/durl/derobeats.tela"><code>/durl/derobeats.tela</code></a></li>
<li><a href="%[1]s/scid/b1e1cba50cbfd8edbb12b01220ffebbece300d4936516a87fc2255fa8e23d8a2/"><code>/scid/&lt;any-index-scid&gt;/</code></a></li>
</ul>
<p>Wallet / EPOCH still needs a local wallet with XSWD (<code>ws://localhost:44326</code>).</p>
<p>Daemon: <code>%[2]s</code></p>
</body></html>`, base, daemon())
}

// writeUnknownApp renders a 404 for a subdomain that matched TELA_HOST_SUFFIX
// but resolves to no known app (typo or unsupported multi-level host). It links
// to the canonical status page and lists the apps that are known.
func writeUnknownApp(w http.ResponseWriter, r *http.Request, app string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	statusURL := ""
	if suffix := strings.TrimPrefix(os.Getenv("TELA_HOST_SUFFIX"), "."); suffix != "" {
		scheme := "https"
		if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
			scheme = p
		} else if r.TLS == nil {
			scheme = "http"
		}
		statusURL = scheme + "://status." + suffix + "/"
	}

	names := make([]string, 0, len(aliases))
	for name := range aliases {
		if !strings.Contains(name, ".") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var list strings.Builder
	for _, name := range names {
		fmt.Fprintf(&list, "<li><code>%s</code></li>", template.HTMLEscapeString(name))
	}
	statusLink := ""
	if statusURL != "" {
		statusLink = fmt.Sprintf(`<p>Gateway status: <a href="%[1]s">%[1]s</a></p>`,
			template.HTMLEscapeString(statusURL))
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>no such TELA app</title>
<style>
body{font-family:system-ui,sans-serif;max-width:42rem;margin:2rem auto;padding:0 1rem;line-height:1.5}
code{background:#f4f4f4;padding:.1em .3em}
</style></head><body>
<h1>404 — no such TELA app</h1>
<p>No TELA app is registered for <code>%s</code>.</p>
%s
<p>Known apps:</p>
<ul>%s</ul>
<p>Any app is reachable directly at <code>/scid/&lt;64-hex-scid&gt;/</code>.</p>
</body></html>`, template.HTMLEscapeString(app), statusLink, list.String())
}

func publicBase(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto + "://" + r.Host
	}
	if r.TLS != nil {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func handleDURL(w http.ResponseWriter, r *http.Request, name string) {
	name = strings.Trim(name, "/")
	if i := strings.IndexByte(name, '/'); i >= 0 {
		name = name[:i]
	}
	key := strings.ToLower(name)
	scid, ok := aliases[key]
	if !ok {
		http.Error(w, "unknown dURL — use /scid/{64-hex}/ for any app, or add an alias", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/scid/"+scid+"/", http.StatusFound)
}

func handleSCID(w http.ResponseWriter, r *http.Request, rest string) {
	rest = strings.TrimPrefix(rest, "/")
	parts := strings.SplitN(rest, "/", 2)
	scid := strings.ToLower(parts[0])
	if !isSCID(scid) {
		http.Error(w, "invalid scid (need 64 hex chars)", http.StatusBadRequest)
		return
	}

	root, err := ensureCloned(scid)
	if err != nil {
		http.Error(w, "clone failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("X-TELA-SCID", scid)
	w.Header().Set("X-TELA-SOURCE", "chain")

	// Serve the cloned app directory. The request may arrive either as
	// /scid/<scid>/<file> (direct path) or as /<file> (vanity host root),
	// so strip the /scid/<scid>/ prefix only when present.
	prefix := "/scid/" + scid + "/"
	upath := r.URL.Path
	if strings.HasPrefix(upath, prefix) {
		upath = strings.TrimPrefix(upath, prefix)
	}
	if upath == "" {
		upath = "/"
	}
	cr := r.Clone(r.Context())
	cr.URL.Path = "/" + strings.TrimPrefix(upath, "/")
	http.FileServer(http.Dir(root)).ServeHTTP(w, cr)
}

func ensureCloned(scid string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if root, ok := roots[scid]; ok {
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			return root, nil
		}
	}

	err := tela.Clone(scid, daemon())
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return "", err
	}

	idx, err := tela.GetINDEXInfo(scid, daemon())
	if err != nil {
		return "", err
	}
	root := filepath.Join(tela.GetClonePath(), idx.DURL)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return "", fmt.Errorf("clone dir missing: %s", root)
	}

	roots[scid] = root
	return root, nil
}
