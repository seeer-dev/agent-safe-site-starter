package main

import (
	"log"
	"net/http"

	"github.com/example/ai-site-starter/server/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("site preview on http://localhost%s", cfg.SiteAddr)
	log.Fatal(http.ListenAndServe(cfg.SiteAddr, http.FileServer(http.Dir("dist"))))
}
