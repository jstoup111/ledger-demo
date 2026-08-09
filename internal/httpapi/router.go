// Package httpapi holds handlers, routing, and JSON + HTML rendering.
package httpapi

import (
	"html/template"
	"net/http"

	"github.com/jstoup111/ledger-demo/web"
)

// NewRouter builds the HTTP handler using the stdlib ServeMux method/pattern
// routing introduced in Go 1.22. No web framework.
//
// Only the HTML page and its stylesheet are wired so far. These routes are added
// during the demo, alongside the domain they expose:
//
//	GET  /api/accounts                    → accounts with derived balances
//	GET  /api/accounts/{id}/transactions  → transactions, newest first
//	POST /api/accounts/{id}/transactions  → post a transaction
//
// JSON errors return a typed code plus a human-readable message.
func NewRouter() (http.Handler, error) {
	page, err := template.ParseFS(web.FS, "index.html.tmpl")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /style.css", http.FileServerFS(web.FS))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, nil); err != nil {
			http.Error(w, "template render failed", http.StatusInternalServerError)
		}
	})

	return mux, nil
}
