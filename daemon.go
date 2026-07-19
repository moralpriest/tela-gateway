package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Public mainnet DERO nodes (Engram-sourced). Foundation node first.
const bundledNodes = "node.derofoundation.org:11012,dero.rabidmining.com:10102,dero-node.net:10102,community-pools.mysrv.cloud:10102"

// BundledNodes returns the public mainnet node list (for entrypoint help text).
func BundledNodes() string { return bundledNodes }

// Default daemon list per target. Overridable via DERO_DAEMON_URLS.
// Local/dev and Docker try a host daemon first; Lambda is public-only.
const defaultLambdaDaemons = bundledNodes

// Daemon failover state.
var (
	daemonMu      sync.RWMutex
	daemonList    []string
	healthyIdx    = -1 // last known-good endpoint index
	unhealthy     = map[int]bool{}
	lastProbe     = map[int]time.Time{}
	probeCacheFor = 30 * time.Second
)

// parseDaemonURLs resolves the configured daemon priority list.
// Accepts DERO_DAEMON_URLS (comma-separated) or legacy DERO_DAEMON_URL (single).
// Strips http(s):// prefixes. Drops localhost when running on Lambda.
func parseDaemonURLs() []string {
	if v := os.Getenv("DERO_DAEMON_URLS"); v != "" {
		return normalizeDaemons(v)
	}
	if v := os.Getenv("DERO_DAEMON_URL"); v != "" {
		return normalizeDaemons(v)
	}

	var base string
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		base = defaultLambdaDaemons
	} else {
		base = "127.0.0.1:10102," + bundledNodes
	}
	return normalizeDaemons(base)
}

func normalizeDaemons(raw string) []string {
	onLambda := os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		p = strings.TrimPrefix(p, "https://")
		p = strings.TrimPrefix(p, "http://")
		if onLambda && isLocalhost(p) {
			// localhost inside Lambda = the sandbox itself (no derod there).
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		out = append(out, "node.derofoundation.org:11012")
	}
	return out
}

func isLocalhost(daemon string) bool {
	h := daemon
	if i := strings.LastIndex(h, ":"); i >= 0 {
		h = h[:i]
	}
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}

// ensureDaemonList initializes the cached daemon list once.
func ensureDaemonList() {
	daemonMu.Lock()
	defer daemonMu.Unlock()
	if daemonList == nil {
		daemonList = parseDaemonURLs()
	}
}

// pickDaemon returns the best available daemon endpoint, probing lazily.
// It prefers the last known-healthy endpoint, otherwise probes the list in
// order (up to one per request, cached for probeCacheFor) and returns the
// first healthy one. Falls back to the first configured endpoint if all
// probes are stale/unhealthy.
func pickDaemon() string {
	ensureDaemonList()

	daemonMu.RLock()
	list := daemonList
	hi := healthyIdx
	daemonMu.RUnlock()

	// Fast path: a known-healthy endpoint that was probed recently.
	if hi >= 0 && hi < len(list) && !isUnhealthy(hi) && recentlyProbed(hi) {
		return list[hi]
	}

	// Probe one endpoint per call, round-robin from last healthy (or 0).
	start := 0
	if hi >= 0 {
		start = (hi + 1) % len(list)
	}
	for i := 0; i < len(list); i++ {
		idx := (start + i) % len(list)
		if isUnhealthy(idx) && recentlyProbed(idx) {
			continue
		}
		if probeHealthy(list[idx]) {
			daemonMu.Lock()
			healthyIdx = idx
			unhealthy[idx] = false
			lastProbe[idx] = time.Now()
			daemonMu.Unlock()
			return list[idx]
		}
		daemonMu.Lock()
		unhealthy[idx] = true
		lastProbe[idx] = time.Now()
		daemonMu.Unlock()
	}

	// All endpoints stale/unhealthy: return last known-good, else first.
	if hi >= 0 && hi < len(list) {
		return list[hi]
	}
	return list[0]
}

func recentlyProbed(idx int) bool {
	daemonMu.RLock()
	defer daemonMu.RUnlock()
	t, ok := lastProbe[idx]
	return ok && time.Since(t) < probeCacheFor
}

func isUnhealthy(idx int) bool {
	daemonMu.RLock()
	defer daemonMu.RUnlock()
	return unhealthy[idx]
}

// probeHealthy does a lightweight DERO.GetInfo JSON-RPC call with a short timeout.
func probeHealthy(daemon string) bool {
	url := "http://" + daemon + "/json_rpc"
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      "tela-gateway-health",
		"method":  "DERO.GetInfo",
		"params":  map[string]any{},
	})

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	var out struct {
		Result struct {
			Height     uint64 `json:"height"`
			TopoHeight uint64 `json:"topoheight"`
			Network    string `json:"network"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	if out.Error != nil {
		return false
	}
	// Require a sane height so we don't trust an empty/erroring node.
	return out.Result.Height > 0 || out.Result.TopoHeight > 0
}

// daemonListString returns the configured list (for health/landing pages).
func daemonListString() string {
	ensureDaemonList()
	daemonMu.RLock()
	defer daemonMu.RUnlock()
	return strings.Join(daemonList, ",")
}
