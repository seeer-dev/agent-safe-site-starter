package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/example/ai-site-starter/server/internal/config"
)

func main() {
	cfg := config.Load()
	if cfg.CFPagesProject == "" {
		log.Fatal("CF_PAGES_PROJECT is required")
	}
	if err := run("go", "run", "./server/tools/render"); err != nil {
		log.Fatal(err)
	}
	if err := run("npx", "wrangler", "pages", "deploy", "dist", "--project-name="+cfg.CFPagesProject); err != nil {
		log.Fatal(err)
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}
