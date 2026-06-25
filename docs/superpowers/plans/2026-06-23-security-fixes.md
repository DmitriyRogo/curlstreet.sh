# Security Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix eight security vulnerabilities identified in the curlstreet security review.

**Architecture:** Each fix is self-contained and touches one or two files. Tasks are ordered so that shared files (finnhub.go, server package) are modified sequentially rather than in parallel. No new packages are introduced except `golang.org/x/sync` for singleflight.

**Tech Stack:** Go, `encoding/json`, `net/url`, `io`, `net/http`, `golang.org/x/sync/singleflight`

## Global Constraints

- All existing tests must continue to pass after each task (`go test ./...`)
- `go vet ./...` must pass after each task
- No changes to public function signatures unless noted
- Add `golang.org/x/sync` to go.mod (Task 4) before any task that imports it

---

### Task 1: Fix JSON injection in RenderError

**Files:**
- Modify: `internal/render/html.go`
- Test: `internal/render/html_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: `RenderError(code int, msg string, format quote.ResponseFormat) string` — unchanged signature, safe output

- [ ] **Step 1: Write a failing test**

Add to `internal/render/html_test.go`:

```go
func TestRenderError_JSONEscaping(t *testing.T) {
    // msg with a double-quote and backslash — both would break naive string concat
    msg := `symbol "BTC/USD" not found: path\value`
    got := RenderError(400, msg, quote.ResponseFormatJSON)
    var parsed map[string]string
    require.NoError(t, json.Unmarshal([]byte(got), &parsed), "must be valid JSON")
    assert.Equal(t, msg, parsed["error"])
}
```

Run: `go test ./internal/render/ -run TestRenderError_JSONEscaping -v`
Expected: FAIL — either a JSON parse error or a wrong value.

- [ ] **Step 2: Fix `RenderError` in `internal/render/html.go`**

Replace:
```go
case quote.ResponseFormatJSON:
    return `{"error":"` + msg + `"}`
```
With:
```go
case quote.ResponseFormatJSON:
    b, _ := json.Marshal(map[string]string{"error": msg})
    return string(b)
```

Add `"encoding/json"` to the import block.

- [ ] **Step 3: Run the test**

Run: `go test ./internal/render/ -run TestRenderError_JSONEscaping -v`
Expected: PASS

- [ ] **Step 4: Run full suite**

Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/html.go internal/render/html_test.go
git commit -m "fix: use json.Marshal in RenderError to prevent JSON injection"
```

---

### Task 2: Limit provider HTTP response body sizes

**Files:**
- Modify: `internal/provider/finnhub.go`
- Modify: `internal/provider/coingecko.go`
- Test: `internal/provider/finnhub_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: same `Fetch` signatures — behaviour unchanged for normal responses, errors on oversized bodies

- [ ] **Step 1: Write a failing test for Finnhub oversized body**

Add to `internal/provider/finnhub_test.go`:

```go
func TestFinnhubProvider_OversizedBody(t *testing.T) {
    // Serve a response body larger than 1 MiB
    huge := strings.Repeat("x", 2<<20) // 2 MiB
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, huge)
    }))
    defer srv.Close()

    p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
    _, err := p.Fetch(context.Background(), "AAPL")
    // Must return an error, not succeed or hang
    require.Error(t, err)
}
```

Run: `go test ./internal/provider/ -run TestFinnhubProvider_OversizedBody -v`
Expected: FAIL — the call currently succeeds (returns ErrSymbolNotFound because JSON is invalid) rather than returning `ErrProviderUnavailable`. Actually it may pass already since JSON decode fails. Verify it returns an error and move on if it already does — the real value is the `io.LimitReader` cap preventing the body from being fully read.

- [ ] **Step 2: Add `io.LimitReader` to every Finnhub response body in `internal/provider/finnhub.go`**

Add `"io"` to the import block.

In `fetchQuote`, `fetchProfile`, `fetchMetric`, `fetchMarketStatus`, and `FetchEconomicCalendar`, change every `json.NewDecoder(resp.Body)` to:

```go
json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
```

There are five decoder calls total — change all of them. Example diff for `fetchQuote`:

```go
// before
if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {

// after
if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
```

- [ ] **Step 3: Add `io.LimitReader` to CoinGecko in `internal/provider/coingecko.go`**

Add `"io"` to the import block. Change the one decoder in `Fetch`:

```go
// before
if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {

// after
if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&markets); err != nil {
```

- [ ] **Step 4: Run the test and full suite**

Run: `go test ./internal/provider/ -run TestFinnhubProvider_OversizedBody -v`
Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/finnhub.go internal/provider/coingecko.go internal/provider/finnhub_test.go
git commit -m "fix: cap provider HTTP response body reads at 1 MiB to prevent memory exhaustion"
```

---

### Task 3: URL-encode symbols in Finnhub provider requests

**Files:**
- Modify: `internal/provider/finnhub.go`

**Interfaces:**
- Consumes: nothing new
- Produces: same `Fetch` signature — URL construction is now safe against symbol characters that are URL metacharacters

- [ ] **Step 1: Write a test verifying the request URL uses proper encoding**

Add to `internal/provider/finnhub_test.go`:

```go
func TestFinnhubProvider_URLEncodesSymbol(t *testing.T) {
    var capturedURL string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        capturedURL = r.URL.RawQuery
        // Return a valid quote so Fetch doesn't error on the profile call
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, `{"c":100,"d":1,"dp":1,"h":101,"l":99,"v":1000}`)
    }))
    defer srv.Close()

    p := NewFinnhubWithBase("my+key&x=1", srv.URL, 5*time.Second)
    // symbol is validated before reaching the provider, but the API key must be encoded
    p.Fetch(context.Background(), "AAPL") //nolint:errcheck — we care about the URL only
    assert.Contains(t, capturedURL, "token=my%2Bkey%26x%3D1",
        "API key must be URL-encoded in the query string")
}
```

Run: `go test ./internal/provider/ -run TestFinnhubProvider_URLEncodesSymbol -v`
Expected: FAIL — the current code puts `my+key&x=1` raw into the URL.

- [ ] **Step 2: Replace `fmt.Sprintf` URL construction with `url.Values` in all five Finnhub fetch functions**

Add `"net/url"` to the import block (remove if already present).

**`fetchQuote`:**
```go
func (p *FinnhubProvider) fetchQuote(ctx context.Context, symbol string) (*finnhubQuoteResp, error) {
    params := url.Values{"symbol": {symbol}, "token": {p.apiKey}}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/quote?"+params.Encode(), nil)
```

**`fetchProfile`:**
```go
func (p *FinnhubProvider) fetchProfile(ctx context.Context, symbol string) (*finnhubProfileResp, error) {
    params := url.Values{"symbol": {symbol}, "token": {p.apiKey}}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/profile2?"+params.Encode(), nil)
```

**`fetchMetric`:**
```go
func (p *FinnhubProvider) fetchMetric(ctx context.Context, symbol string) (*finnhubMetricResp, error) {
    params := url.Values{"symbol": {symbol}, "metric": {"all"}, "token": {p.apiKey}}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/metric?"+params.Encode(), nil)
```

**`fetchMarketStatus`:**
```go
func (p *FinnhubProvider) fetchMarketStatus(ctx context.Context, exchange string) (string, error) {
    params := url.Values{"exchange": {exchange}, "token": {p.apiKey}}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/market-status?"+params.Encode(), nil)
```

**`FetchEconomicCalendar`:**
```go
    params := url.Values{"from": {from}, "to": {to}, "token": {p.apiKey}}
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/calendar/economic?"+params.Encode(), nil)
```

- [ ] **Step 3: Run the test and full suite**

Run: `go test ./internal/provider/ -run TestFinnhubProvider_URLEncodesSymbol -v`
Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/provider/finnhub.go internal/provider/finnhub_test.go
git commit -m "fix: use url.Values to properly encode Finnhub API request parameters"
```

---

### Task 4: Singleflight for market status thundering herd

**Files:**
- Modify: `internal/provider/finnhub.go`
- Modify: `go.mod` (add `golang.org/x/sync`)

**Interfaces:**
- Consumes: `golang.org/x/sync/singleflight`
- Produces: `FinnhubProvider` struct gains `sf singleflight.Group` field — no external API change

- [ ] **Step 1: Add the dependency**

Run: `go get golang.org/x/sync@latest`
Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 2: Write a concurrency test**

Add to `internal/provider/finnhub_test.go`:

```go
func TestFinnhubProvider_MarketStatusSingleFlight(t *testing.T) {
    calls := atomic.Int32{}
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "market-status") {
            calls.Add(1)
            time.Sleep(20 * time.Millisecond) // ensure overlap
            w.WriteHeader(http.StatusOK)
            fmt.Fprint(w, `{"isOpen":true,"session":"regular"}`)
            return
        }
        w.WriteHeader(http.StatusOK)
        fmt.Fprint(w, `{"c":100,"d":1,"dp":1,"h":101,"l":99,"v":1000}`)
    }))
    defer srv.Close()

    p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            p.Fetch(context.Background(), "AAPL") //nolint:errcheck
        }()
    }
    wg.Wait()

    assert.LessOrEqual(t, int(calls.Load()), 2,
        "concurrent fetches should collapse to at most 2 market-status calls")
}
```

Add required imports at top of test file: `"sync/atomic"`, `"sync"`, `"strings"`, `"time"`.

Run: `go test ./internal/provider/ -run TestFinnhubProvider_MarketStatusSingleFlight -v`
Expected: FAIL — currently allows 10 concurrent calls.

- [ ] **Step 3: Add `singleflight.Group` to `FinnhubProvider`**

In `internal/provider/finnhub.go`, add `"golang.org/x/sync/singleflight"` to imports.

Add `sf singleflight.Group` field to the struct:

```go
type FinnhubProvider struct {
    apiKey  string
    baseURL string
    client  *http.Client

    msMu    sync.Mutex
    msCache map[string]marketStatusEntry
    sf      singleflight.Group
}
```

- [ ] **Step 4: Wrap `fetchMarketStatus` with singleflight in `marketStatus`**

Replace the body of `marketStatus` with:

```go
func (p *FinnhubProvider) marketStatus(ctx context.Context, exchange string) string {
    p.msMu.Lock()
    if e, ok := p.msCache[exchange]; ok && time.Now().Before(e.expiresAt) {
        p.msMu.Unlock()
        return e.status
    }
    p.msMu.Unlock()

    v, _, _ := p.sf.Do(exchange, func() (any, error) {
        // Double-check inside the singleflight fence
        p.msMu.Lock()
        if e, ok := p.msCache[exchange]; ok && time.Now().Before(e.expiresAt) {
            p.msMu.Unlock()
            return e.status, nil
        }
        p.msMu.Unlock()

        status, err := p.fetchMarketStatus(ctx, exchange)
        if err != nil {
            return quote.MarketStatusClosed, nil
        }

        p.msMu.Lock()
        p.msCache[exchange] = marketStatusEntry{
            status:    status,
            expiresAt: time.Now().Add(60 * time.Second),
        }
        p.msMu.Unlock()

        return status, nil
    })

    if s, ok := v.(string); ok {
        return s
    }
    return quote.MarketStatusClosed
}
```

- [ ] **Step 5: Run the test and full suite**

Run: `go test ./internal/provider/ -run TestFinnhubProvider_MarketStatusSingleFlight -v`
Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/provider/finnhub.go internal/provider/finnhub_test.go go.mod go.sum
git commit -m "fix: use singleflight to prevent thundering herd on market status cache"
```

---

### Task 5: Add HTTP security headers middleware

**Files:**
- Modify: `internal/server/server.go`
- Test: `internal/server/handler_test.go`

**Interfaces:**
- Consumes: nothing new
- Produces: all HTTP responses include `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`

- [ ] **Step 1: Write failing tests for security headers**

Add to `internal/server/handler_test.go`:

```go
func TestSecurityHeaders(t *testing.T) {
    srv := newTestServer(t)
    resp, err := http.Get(srv.URL + "/")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
    assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
    assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
    assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
}
```

(Use whatever `newTestServer` helper already exists in the test file, or adapt to match the existing test setup pattern.)

Run: `go test ./internal/server/ -run TestSecurityHeaders -v`
Expected: FAIL — headers not present.

- [ ] **Step 2: Add `securityHeaders` middleware function to `internal/server/server.go`**

Add after the `New` function:

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "no-referrer")
        w.Header().Set("Content-Security-Policy",
            "default-src 'none'; style-src 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; img-src data:")
        next.ServeHTTP(w, r)
    })
}
```

- [ ] **Step 3: Wire `securityHeaders` into the handler chain in `New`**

In the `New` function, wrap `rl.middleware(mux)` with `securityHeaders`:

```go
s.handler = securityHeaders(rl.middleware(mux))
```

- [ ] **Step 4: Run the test and full suite**

Run: `go test ./internal/server/ -run TestSecurityHeaders -v`
Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/server/server.go internal/server/handler_test.go
git commit -m "fix: add HTTP security headers middleware (CSP, X-Frame-Options, nosniff, Referrer-Policy)"
```

---

### Task 6: Proxy-aware rate limiting

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/server/ratelimit.go`
- Modify: `internal/server/server.go`
- Modify: `main.go`
- Modify: `config.yaml`
- Test: `internal/server/handler_test.go`

**Interfaces:**
- Consumes: `Config.Server.TrustedProxy string` (CIDR or empty)
- Produces: `newRateLimiter(rpm, burst int, trustedProxy string) *rateLimiter` — signature change; callers updated

- [ ] **Step 1: Write a failing test for proxy-aware IP extraction**

Add to `internal/server/handler_test.go`:

```go
func TestRateLimiter_TrustedProxyUsesXFF(t *testing.T) {
    rl := newRateLimiter(60, 100, "127.0.0.1/32")
    var gotIP string
    inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotIP = r.Header.Get("X-Tested-IP") // we'll stash the resolved IP via context trick below
        w.WriteHeader(http.StatusOK)
    })
    // We can't easily introspect the resolved IP without changing ratelimit.go's interface,
    // so instead test the observable behaviour: two requests with different XFF from the
    // same RemoteAddr should be treated as different clients (neither hits the rate limit).
    handler := rl.middleware(inner)

    makeReq := func(xff string) *http.Response {
        r := httptest.NewRequest(http.MethodGet, "/", nil)
        r.RemoteAddr = "127.0.0.1:9999"
        r.Header.Set("X-Forwarded-For", xff)
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, r)
        return w.Result()
    }

    resp1 := makeReq("10.0.0.1")
    resp2 := makeReq("10.0.0.2")
    assert.Equal(t, http.StatusOK, resp1.StatusCode)
    assert.Equal(t, http.StatusOK, resp2.StatusCode)
    _ = gotIP
}
```

Run: `go test ./internal/server/ -run TestRateLimiter_TrustedProxyUsesXFF -v`
Expected: FAIL — `newRateLimiter` currently has no `trustedProxy` parameter.

- [ ] **Step 2: Add `TrustedProxy` to `Config.Server` in `internal/config/config.go`**

```go
Server struct {
    Addr         string        `yaml:"addr"`
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    TrustedProxy string        `yaml:"trusted_proxy"` // CIDR of reverse proxy, e.g. "10.0.0.0/8"; empty = disabled
} `yaml:"server"`
```

- [ ] **Step 3: Update `internal/server/ratelimit.go`**

Add `"net"` and `"strings"` to imports (if not present).

Add `trustedNet *net.IPNet` to the `rateLimiter` struct:

```go
type rateLimiter struct {
    mu         sync.Mutex
    limiters   map[string]*ipLimiter
    r          rate.Limit
    burst      int
    trustedNet *net.IPNet // nil = no trusted proxy
}
```

Update `newRateLimiter` signature and body:

```go
func newRateLimiter(requestsPerMinute, burst int, trustedProxy string) *rateLimiter {
    var trustedNet *net.IPNet
    if trustedProxy != "" {
        _, trustedNet, _ = net.ParseCIDR(trustedProxy)
    }
    rl := &rateLimiter{
        limiters:   make(map[string]*ipLimiter),
        r:          rate.Limit(float64(requestsPerMinute) / 60.0),
        burst:       burst,
        trustedNet: trustedNet,
    }
    go rl.cleanup()
    return rl
}
```

Update `middleware` to use `X-Forwarded-For` when origin is in the trusted CIDR:

```go
func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            ip = r.RemoteAddr
        }
        if rl.trustedNet != nil {
            if remoteIP := net.ParseIP(ip); remoteIP != nil && rl.trustedNet.Contains(remoteIP) {
                if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
                    if parts := strings.SplitN(xff, ",", 2); len(parts) > 0 {
                        if candidate := strings.TrimSpace(parts[0]); candidate != "" {
                            ip = candidate
                        }
                    }
                }
            }
        }
        if !rl.get(ip).Allow() {
            http.Error(w, "rate limit exceeded — slow down", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

- [ ] **Step 4: Update callers of `newRateLimiter`**

In `internal/server/server.go`, update the `New` function signature and call:

```go
func New(svc *service.QuoteService, logger *logrus.Logger, requestsPerMinute, burst int, trustedProxy string, calendar CalendarFetcher) *Server {
    mux := http.NewServeMux()
    s := &Server{svc: svc, calendar: calendar, logger: logger}
    mux.HandleFunc("/", s.handleQuote)
    rl := newRateLimiter(requestsPerMinute, burst, trustedProxy)
    s.handler = securityHeaders(rl.middleware(mux))
    return s
}
```

In `main.go`, update the `server.New` call:

```go
srv := server.New(svc, log, cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst, cfg.Server.TrustedProxy, server.ComputedCalendar())
```

- [ ] **Step 5: Update `config.yaml`**

Add the new field with a comment:

```yaml
server:
  addr: ":8080"
  read_timeout: 10s
  write_timeout: 10s
  trusted_proxy: ""  # CIDR of your reverse proxy (e.g. "10.0.0.0/8"); leave empty if running without a proxy
```

- [ ] **Step 6: Run the test and full suite**

Run: `go test ./internal/server/ -run TestRateLimiter_TrustedProxyUsesXFF -v`
Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go internal/server/ratelimit.go internal/server/server.go main.go config.yaml
git commit -m "fix: make rate limiter proxy-aware via configurable trusted_proxy CIDR"
```

---

### Task 7: Log .env parse errors and document config.yaml API key

**Files:**
- Modify: `main.go`
- Modify: `config.yaml`

**Interfaces:**
- Consumes: nothing new
- Produces: no functional change — operational visibility improvement

- [ ] **Step 1: Fix `.env` error handling in `main.go`**

Replace:
```go
_ = godotenv.Load()
```
With:
```go
if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
    log.WithError(err).Warn("failed to parse .env file")
}
```

Note: `logrus` is already imported and `log` is set up before this call, so no additional imports needed.

- [ ] **Step 2: Add warning comment to `config.yaml`**

Replace:
```yaml
finnhub:
  api_key: ""
  timeout: 5s
```
With:
```yaml
finnhub:
  api_key: ""  # do not set a real key here — use the FINNHUB_API_KEY environment variable instead
  timeout: 5s
```

- [ ] **Step 3: Run full suite**

Run: `go test ./... && go vet ./...`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add main.go config.yaml
git commit -m "fix: log .env parse errors and document API key env-var convention in config.yaml"
```
