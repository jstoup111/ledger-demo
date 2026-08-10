// Package acceptance holds the end-to-end acceptance specs for the base ledger.
//
// These specs drive the REAL production entry points — the ./cmd/server binary's
// "seed" and "serve" subcommands — over real HTTP against a file-backed SQLite
// database inside t.TempDir(). Nothing here imports internal packages, so the
// specs stay valid while the implementation tasks reshape internal signatures,
// and a failure here always means the assembled demo is wrong rather than that a
// helper moved.
//
// Conventions this file inherits from the project:
//   - stdlib testing only, table-driven, no testify and no mocking library
//   - no time.Sleep anywhere (NFR-4); Story 6 permits one system-clock call
//     site in the repository, inside SystemClock.
//     Readiness therefore uses a deadline-bounded dial loop over time.After.
//   - the default ./ledger.db is never touched (conflict-check F3)
package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// readyTimeout bounds how long a freshly started server may take to accept a
// connection before the spec gives up and reports the server's own output.
const readyTimeout = 15 * time.Second

// serverBin is the compiled ./cmd/server binary, built once for the package.
var serverBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ledger-acceptance-bin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "acceptance: creating temp dir: %v\n", err)
		os.Exit(1)
	}

	serverBin = filepath.Join(dir, "ledger-server")
	build := exec.Command("go", "build", "-o", serverBin, "./cmd/server")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "acceptance: building ./cmd/server: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// --- wire types (the contract in .docs/decisions/api-response-contract.md) ---

type apiAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BalanceCents int64  `json:"balance_cents"`
}

type apiTransaction struct {
	ID          string `json:"id"`
	AccountID   string `json:"account_id"`
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// --- the app under test ---

type app struct {
	t      *testing.T
	dbPath string
	base   string
	client *http.Client
}

// newApp seeds a pristine database in a temporary directory and serves it,
// exactly as `make reset && make dev` does.
func newApp(t *testing.T) *app {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "acceptance.db")
	seedDB(t, dbPath)
	base, _ := startServer(t, dbPath)

	return newAppAt(t, dbPath, base)
}

func newAppAt(t *testing.T, dbPath, base string) *app {
	t.Helper()
	return &app{
		t:      t,
		dbPath: dbPath,
		base:   base,
		// Redirects are the subject of several specs, so they are never followed
		// implicitly — a spec follows one deliberately via app.get.
		client: &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// seedDB runs the real `seed` subcommand against dbPath.
func seedDB(t *testing.T, dbPath string) {
	t.Helper()

	cmd := exec.Command(serverBin, "seed")
	cmd.Env = append(os.Environ(), "LEDGER_DB_PATH="+dbPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("seed %s: %v\n%s", dbPath, err, out)
	}
}

// exportCSV runs the real `export` subcommand against dbPath while keeping its
// data output separate from diagnostics.
func exportCSV(t *testing.T, dbPath, accountID string) (stdout, stderr []byte, err error) {
	t.Helper()

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd := exec.Command(serverBin, "export", accountID)
	cmd.Env = append(os.Environ(), "LEDGER_DB_PATH="+dbPath)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err = cmd.Run()
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), err
}

// startServer runs the real `serve` subcommand on a free port. The returned stop
// function is idempotent and is also registered as test cleanup.
func startServer(t *testing.T, dbPath string) (base string, stop func()) {
	t.Helper()

	port := freePort(t)
	logs := &safeBuffer{}

	cmd := exec.Command(serverBin, "serve")
	cmd.Env = append(os.Environ(), "LEDGER_DB_PATH="+dbPath, "PORT="+port)
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		t.Fatalf("serve %s on port %s: %v", dbPath, port, err)
	}

	var once sync.Once
	stop = func() {
		once.Do(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
	}
	t.Cleanup(stop)

	base = "http://127.0.0.1:" + port
	waitReady(t, port, logs)
	return base, stop
}

func waitReady(t *testing.T, port string, logs *safeBuffer) {
	t.Helper()

	addr := net.JoinHostPort("127.0.0.1", port)
	timeout := time.After(readyTimeout)
	for {
		select {
		case <-timeout:
			t.Fatalf("server did not accept a connection on %s within %s; server output:\n%s",
				addr, readyTimeout, logs.String())
		default:
		}

		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return
		}
	}
}

func freePort(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a free port: %v", err)
	}
	defer l.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("splitting %q: %v", l.Addr().String(), err)
	}
	return port
}

// safeBuffer collects child-process output without racing the reader.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// --- request helpers ---

type response struct {
	status   int
	header   http.Header
	body     string
	location string
}

func (a *app) do(req *http.Request) response {
	a.t.Helper()

	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		a.t.Fatalf("reading body of %s %s: %v", req.Method, req.URL, err)
	}

	return response{
		status:   res.StatusCode,
		header:   res.Header,
		body:     string(body),
		location: res.Header.Get("Location"),
	}
}

func (a *app) request(method, path, contentType, body string) response {
	a.t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, a.base+path, reader)
	if err != nil {
		a.t.Fatalf("building %s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return a.do(req)
}

func (a *app) get(path string) response {
	a.t.Helper()
	return a.request(http.MethodGet, path, "", "")
}

func (a *app) postJSON(path, body string) response {
	a.t.Helper()
	return a.request(http.MethodPost, path, "application/json", body)
}

func (a *app) postForm(path string, values url.Values) response {
	a.t.Helper()
	return a.request(http.MethodPost, path, "application/x-www-form-urlencoded", values.Encode())
}

// --- decoding helpers ---

func (a *app) accounts() []apiAccount {
	a.t.Helper()

	res := a.get("/api/accounts")
	if res.status != http.StatusOK {
		a.t.Fatalf("GET /api/accounts status = %d, want 200; body:\n%s", res.status, res.body)
	}

	var accounts []apiAccount
	if err := json.Unmarshal([]byte(res.body), &accounts); err != nil {
		a.t.Fatalf("GET /api/accounts body is not a JSON array of accounts: %v; body:\n%s", err, res.body)
	}
	return accounts
}

func (a *app) balanceCents(accountID string) int64 {
	a.t.Helper()

	for _, acct := range a.accounts() {
		if acct.ID == accountID {
			return acct.BalanceCents
		}
	}
	a.t.Fatalf("account %q is not present in GET /api/accounts", accountID)
	return 0
}

func (a *app) transactions(accountID string) []apiTransaction {
	a.t.Helper()

	path := "/api/accounts/" + url.PathEscape(accountID) + "/transactions"
	res := a.get(path)
	if res.status != http.StatusOK {
		a.t.Fatalf("GET %s status = %d, want 200; body:\n%s", path, res.status, res.body)
	}

	var txs []apiTransaction
	if err := json.Unmarshal([]byte(res.body), &txs); err != nil {
		a.t.Fatalf("GET %s body is not a JSON array of transactions: %v; body:\n%s", path, err, res.body)
	}
	return txs
}

func (a *app) transactionCount(accountID string) int {
	a.t.Helper()
	return len(a.transactions(accountID))
}

func decodeJSON(t *testing.T, body string, into any) {
	t.Helper()

	if err := json.Unmarshal([]byte(body), into); err != nil {
		t.Fatalf("body is not the JSON shape the contract documents: %v; body:\n%s", err, body)
	}
}

func decodeError(t *testing.T, res response) apiError {
	t.Helper()

	var payload apiError
	if err := json.Unmarshal([]byte(res.body), &payload); err != nil {
		t.Fatalf("rejection body is not the documented error shape: %v; body:\n%s", err, res.body)
	}
	return payload
}

// --- assertion helpers ---

// formatDollars renders integer cents the way the page must, with no float
// arithmetic on the path: $1,283.50 and -$42.50.
func formatDollars(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}

	dollars := cents / 100
	remainder := cents % 100

	grouped := fmt.Sprintf("%d", dollars)
	for i := len(grouped) - 3; i > 0; i -= 3 {
		grouped = grouped[:i] + "," + grouped[i:]
	}

	return fmt.Sprintf("%s$%s.%02d", sign, grouped, remainder)
}

func mustContain(t *testing.T, body, want, what string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Errorf("%s: body does not contain %q; body:\n%s", what, want, body)
	}
}

func mustNotContain(t *testing.T, body, unwanted, what string) {
	t.Helper()
	if strings.Contains(body, unwanted) {
		t.Errorf("%s: body unexpectedly contains %q; body:\n%s", what, unwanted, body)
	}
}
