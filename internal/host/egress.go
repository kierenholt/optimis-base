package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	DefaultEgressTimeoutMS = 8000
	maxEgressRedirects     = 10
	maxEgressBodyBytes     = 4 * 1024 * 1024
)

type MockEgressRule struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	RedirectTo string            `json:"redirect_to,omitempty"`
	DelayMS    int               `json:"delay_ms,omitempty"`
}

type hostEgressState struct {
	mu                   sync.RWMutex
	enableRealHTTPEgress bool
	mockRules            map[string]MockEgressRule
}

var (
	egressStatesMu sync.Mutex
	egressStates   = make(map[*Host]*hostEgressState)
)

func (h *Host) getEgressState() *hostEgressState {
	egressStatesMu.Lock()
	defer egressStatesMu.Unlock()
	st, ok := egressStates[h]
	if !ok {
		enableReal := false
		if raw := strings.TrimSpace(os.Getenv("ENABLE_REAL_HTTP_EGRESS")); strings.EqualFold(raw, "true") {
			enableReal = true
		}
		st = &hostEgressState{
			enableRealHTTPEgress: enableReal,
			mockRules:            make(map[string]MockEgressRule),
		}
		egressStates[h] = st
	}
	return st
}

func (h *Host) ConfigureEgressMode(enableRealHTTPEgress bool) {
	st := h.getEgressState()
	st.mu.Lock()
	st.enableRealHTTPEgress = enableRealHTTPEgress
	st.mu.Unlock()
	h.logInfo(fmt.Sprintf("Configured host HTTP egress mode: ENABLE_REAL_HTTP_EGRESS=%v", enableRealHTTPEgress))
}

func (h *Host) IsRealHTTPEgressEnabled() bool {
	st := h.getEgressState()
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.enableRealHTTPEgress
}

func mockRuleKey(method string, rawURL string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(rawURL)
}

func (h *Host) RegisterMockEgressRule(rule MockEgressRule) error {
	method := strings.ToUpper(strings.TrimSpace(rule.Method))
	if method == "" {
		return errors.New("missing mandatory mock egress field: method")
	}
	targetURL := strings.TrimSpace(rule.URL)
	if targetURL == "" {
		return errors.New("missing mandatory mock egress field: url")
	}
	if rule.Status <= 0 {
		if strings.TrimSpace(rule.RedirectTo) != "" {
			rule.Status = http.StatusFound
		} else {
			rule.Status = http.StatusOK
		}
	}
	rule.Method = method
	rule.URL = targetURL

	st := h.getEgressState()
	st.mu.Lock()
	st.mockRules[mockRuleKey(method, targetURL)] = rule
	st.mu.Unlock()
	return nil
}

func (h *Host) ClearMockEgressRules() {
	st := h.getEgressState()
	st.mu.Lock()
	st.mockRules = make(map[string]MockEgressRule)
	st.mu.Unlock()
}

func ResolveEgressTimeoutMS(configuredMS int) int {
	if configuredMS <= 0 {
		return DefaultEgressTimeoutMS
	}
	return configuredMS
}

func isLoopbackHost(hostPort string) bool {
	hostname := hostPort
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		hostname = h
	}
	lower := strings.ToLower(strings.TrimSpace(hostname))
	return lower == "127.0.0.1" || lower == "localhost" || lower == "::1"
}

func isDomainAllowed(targetURL string, allowed []string) bool {
	trimmedTarget := strings.TrimSpace(targetURL)
	if trimmedTarget == "" || len(allowed) == 0 {
		return false
	}
	parsedTarget, err := url.Parse(trimmedTarget)
	if err != nil || parsedTarget.Scheme == "" || parsedTarget.Host == "" {
		return false
	}

	targetScheme := strings.ToLower(parsedTarget.Scheme)
	if targetScheme != "https" && !(targetScheme == "http" && isLoopbackHost(parsedTarget.Host)) {
		return false
	}

	for _, origin := range allowed {
		trimmedOrigin := strings.TrimSpace(origin)
		if trimmedOrigin == "" {
			continue
		}
		parsedOrigin, err := url.Parse(trimmedOrigin)
		if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
			continue
		}
		if !strings.EqualFold(parsedTarget.Scheme, parsedOrigin.Scheme) {
			continue
		}
		if !strings.EqualFold(parsedTarget.Host, parsedOrigin.Host) {
			continue
		}
		if parsedOrigin.Path != "" && parsedOrigin.Path != "/" {
			if !strings.HasPrefix(parsedTarget.Path, parsedOrigin.Path) {
				continue
			}
		}
		return true
	}
	return false
}

type mockEgressRoundTripper struct {
	host *Host
}

func (m *mockEgressRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL != nil && strings.EqualFold(req.URL.Host, "collect.dfe.gov.uk") {
		return m.host.handleMockDfeCollectRoundTrip(req)
	}

	st := m.host.getEgressState()
	key := mockRuleKey(req.Method, req.URL.String())

	st.mu.RLock()
	rule, exists := st.mockRules[key]
	st.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("MOCK_EGRESS_UNCONFIGURED_URL: no mock egress rule registered for %s %s", req.Method, req.URL.String())
	}

	if rule.DelayMS > 0 {
		select {
		case <-time.After(time.Duration(rule.DelayMS) * time.Millisecond):
		case <-req.Context().Done():
			return nil, fmt.Errorf("EGRESS_TIMEOUT: %w", req.Context().Err())
		}
	}

	hdr := make(http.Header)
	for k, v := range rule.Headers {
		hdr.Set(k, v)
	}
	if strings.TrimSpace(rule.RedirectTo) != "" {
		hdr.Set("Location", strings.TrimSpace(rule.RedirectTo))
	}

	return &http.Response{
		StatusCode: rule.Status,
		Status:     fmt.Sprintf("%d %s", rule.Status, http.StatusText(rule.Status)),
		Header:     hdr,
		Body:       io.NopCloser(strings.NewReader(rule.Body)),
		Request:    req,
	}, nil
}

func (h *Host) buildLuaHTTPEgressTable(
	L *lua.LState,
	execCtx context.Context,
	plugin *LoadedPlugin,
	writeTx **sql.Tx,
) *lua.LTable {
	httpTbl := L.NewTable()

	L.SetField(httpTbl, "get", L.NewFunction(func(L *lua.LState) int {
		return h.executeLuaHTTPRequest(L, execCtx, plugin, writeTx, http.MethodGet, "ctx.http.get")
	}))

	L.SetField(httpTbl, "post", L.NewFunction(func(L *lua.LState) int {
		return h.executeLuaHTTPRequest(L, execCtx, plugin, writeTx, http.MethodPost, "ctx.http.post")
	}))

	return httpTbl
}

func (h *Host) executeLuaHTTPRequest(
	L *lua.LState,
	execCtx context.Context,
	plugin *LoadedPlugin,
	writeTx **sql.Tx,
	httpMethod string,
	methodLabel string,
) int {
	// 1. Structural Safety Invariant (§9.2 & §10.1): Disallowed inside open SQLite write transactions
	if *writeTx != nil {
		_ = (*writeTx).Rollback()
		*writeTx = nil
		h.recordDiagnostic(plugin.Manifest.ID, methodLabel, "EGRESS_WRITE_TX_HALT", 0)
		h.logError(fmt.Sprintf("Halted open WriteDB transaction in %s: %s disallowed inside an open SQLite write transaction", plugin.Manifest.ID, methodLabel))
		L.RaiseError("%s disallowed inside an open SQLite write transaction", methodLabel)
		return 0
	}

	// 2. Validate target URL against manifest.json egress.allowed_domains
	targetURL := strings.TrimSpace(L.OptString(1, ""))
	if targetURL == "" {
		L.RaiseError("%s requires a non-empty target URL", methodLabel)
		return 0
	}

	allowedDomains := plugin.Manifest.Egress.AllowedDomains
	if !isDomainAllowed(targetURL, allowedDomains) {
		h.recordDiagnostic(plugin.Manifest.ID, methodLabel, "EGRESS_DOMAIN_BLOCKED", 0)
		h.logError(fmt.Sprintf("Blocked outbound %s from %s to unapproved domain: %s", httpMethod, plugin.Manifest.ID, targetURL))
		L.RaiseError("egress domain not permitted by manifest.json: %s", targetURL)
		return 0
	}

	// 3. Extract optional headers table (arg 2) and request body string (arg 3 for POST)
	reqHeaders := make(map[string]string)
	if L.GetTop() >= 2 && L.Get(2) != lua.LNil {
		hdrTbl, ok := L.Get(2).(*lua.LTable)
		if !ok {
			L.RaiseError("%s headers argument must be a Lua table", methodLabel)
			return 0
		}
		hdrTbl.ForEach(func(k, v lua.LValue) {
			key := strings.TrimSpace(k.String())
			if key != "" {
				reqHeaders[key] = v.String()
			}
		})
	}

	var reqBodyReader io.Reader
	if httpMethod == http.MethodPost {
		bodyStr := L.OptString(3, "")
		reqBodyReader = strings.NewReader(bodyStr)
	}

	// 4. Enforce egress.timeout_ms (fallback 8000ms per §10.1)
	timeoutMS := ResolveEgressTimeoutMS(plugin.Manifest.Egress.TimeoutMS)
	timeoutDur := time.Duration(timeoutMS) * time.Millisecond
	reqCtx, cancel := context.WithTimeout(execCtx, timeoutDur)
	defer cancel()

	var transport http.RoundTripper = http.DefaultTransport
	if !h.IsRealHTTPEgressEnabled() {
		transport = &mockEgressRoundTripper{host: h}
	}

	// 5. Construct http.Client with custom CheckRedirect domain whitelist verification
	client := &http.Client{
		Transport: transport,
		Timeout:   timeoutDur,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxEgressRedirects {
				return fmt.Errorf("EGRESS_MAX_REDIRECTS_EXCEEDED: stopped after %d redirects", maxEgressRedirects)
			}
			redirectURL := req.URL.String()
			if !isDomainAllowed(redirectURL, allowedDomains) {
				h.recordDiagnostic(plugin.Manifest.ID, methodLabel, "EGRESS_REDIRECT_BLOCKED", 0)
				h.logError(fmt.Sprintf("Blocked outbound redirect in %s to unapproved domain: %s", plugin.Manifest.ID, redirectURL))
				return fmt.Errorf("egress redirect target domain not permitted by manifest.json: %s", redirectURL)
			}
			if len(via) > 0 {
				for k, vals := range via[0].Header {
					if req.Header.Get(k) == "" {
						for _, v := range vals {
							req.Header.Add(k, v)
						}
					}
				}
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(reqCtx, httpMethod, targetURL, reqBodyReader)
	if err != nil {
		L.RaiseError("%s failed to build HTTP request: %s", methodLabel, err.Error())
		return 0
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "Client.Timeout") ||
			strings.Contains(err.Error(), "EGRESS_TIMEOUT") {
			h.recordDiagnostic(plugin.Manifest.ID, methodLabel, "EGRESS_TIMEOUT", 0)
			h.logError(fmt.Sprintf("Outbound %s in %s timed out after %dms: %v", httpMethod, plugin.Manifest.ID, timeoutMS, err))
			L.RaiseError("EGRESS_TIMEOUT: %s timed out after %dms: %s", methodLabel, timeoutMS, err.Error())
			return 0
		}
		h.recordDiagnostic(plugin.Manifest.ID, methodLabel, "EGRESS_REQUEST_FAILED", 0)
		h.logError(fmt.Sprintf("Outbound %s in %s failed: %v", httpMethod, plugin.Manifest.ID, err))
		L.RaiseError("%s failed: %s", methodLabel, err.Error())
		return 0
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxEgressBodyBytes))
	if readErr != nil {
		L.RaiseError("%s failed reading response body: %s", methodLabel, readErr.Error())
		return 0
	}

	finalURL := targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	respHeadersTbl := L.NewTable()
	for k, vals := range resp.Header {
		joined := strings.Join(vals, ", ")
		L.SetField(respHeadersTbl, strings.ToLower(k), lua.LString(joined))
		L.SetField(respHeadersTbl, k, lua.LString(joined))
	}

	respTbl := L.NewTable()
	L.SetField(respTbl, "status", lua.LNumber(resp.StatusCode))
	L.SetField(respTbl, "body", lua.LString(string(bodyBytes)))
	L.SetField(respTbl, "headers", respHeadersTbl)
	L.SetField(respTbl, "final_url", lua.LString(finalURL))
	L.Push(respTbl)
	return 1
}

func (h *Host) RegisterEgressTestRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/test/egress-mode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EnableRealHTTPEgress *bool `json:"enable_real_http_egress"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EnableRealHTTPEgress == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": "missing mandatory boolean field: enable_real_http_egress",
			})
			return
		}
		h.ConfigureEgressMode(*body.EnableRealHTTPEgress)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":                      true,
			"enable_real_http_egress": *body.EnableRealHTTPEgress,
		})
	})

	mux.HandleFunc("POST /api/test/egress-mock", func(w http.ResponseWriter, r *http.Request) {
		var rule MockEgressRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": fmt.Sprintf("invalid JSON body: %v", err),
			})
			return
		}
		if err := h.RegisterMockEgressRule(rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":         false,
				"error_code": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"ok":   true,
			"rule": rule,
		})
	})

	mux.HandleFunc("DELETE /api/test/egress-mock", func(w http.ResponseWriter, r *http.Request) {
		h.ClearMockEgressRules()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"cleared": true,
		})
	})
}
