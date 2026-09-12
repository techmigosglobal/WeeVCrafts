// WeeVCrafts customer storefront preview.
//
// This entry point intentionally serves the isolated UI fixture store. The
// existing cmd/res2 composition remains the backend-connected application and
// can be wired to these view models in a later milestone.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/wecratfs/commerce/internal/web/handlers"
	"github.com/wecratfs/commerce/internal/web/middleware"
)

func main() {
	mode := middleware.ParseMode(os.Getenv("WEB_UI_MODE"))
	if mode != middleware.ModeMock {
		log.Fatal("cmd/web is the UI-first preview; set WEB_UI_MODE=mock (live integration is deferred)")
	}

	addr := os.Getenv("WEB_UI_ADDR")
	if addr == "" {
		addr = ":8090"
	}
	h := middleware.PreviewHeaders(handlers.NewMockHandler())
	log.Printf("WeeVCrafts UI preview listening on %s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatal(err)
	}
}
