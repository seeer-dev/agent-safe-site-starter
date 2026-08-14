package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/example/ai-site-starter/server/internal/config"
)

func main() {
	cfg := config.Load()

	// Step 1: Render static output to staging, then atomically promote.
	// On render failure, the existing dist is preserved and we abort.
	if err := run("go", "run", "./server/tools/render"); err != nil {
		log.Fatalf("render failed (dist preserved): %v", err)
	}
	log.Println("render completed, dist promoted")

	// Step 2: Trigger Cloudflare Deploy Hook if CF_DEPLOY_HOOK_URL is set.
	// GATE-006: This is a trigger-only flow. We do NOT perform a Direct
	// Upload via wrangler — the Deploy Hook tells Cloudflare Pages to
	// re-deploy from the connected Git branch. Mixing Direct Upload and
	// Deploy Hook would cause a double-deploy race.
	//
	// The receipt includes the HTTP status, response body, and timestamp
	// so the operator can verify the trigger was accepted. If no hook URL
	// is configured, we log that the hook was skipped.
	if cfg.CFDeployHookURL != "" {
		receipt, err := triggerDeployHook(cfg.CFDeployHookURL)
		if err != nil {
			log.Fatalf("deploy hook trigger failed: %v", err)
		}
		log.Printf("deploy hook receipt: %s", receipt)
	} else {
		log.Println("deploy hook skipped (CF_DEPLOY_HOOK_URL not set)")
	}
}

// triggerDeployHook POSTs to the Cloudflare Pages Deploy Hook URL and
// returns a truthful receipt string. The receipt includes the timestamp,
// HTTP status code, and response body so the operator can verify the
// trigger was accepted.
func triggerDeployHook(hookURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, hookURL, bytes.NewReader(nil))
	if err != nil {
		return "", fmt.Errorf("create deploy hook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deploy hook request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read deploy hook response: %w", err)
	}

	receipt := fmt.Sprintf(`{"triggered_at":"%s","status":%d,"body":%q}`,
		time.Now().UTC().Format(time.RFC3339), resp.StatusCode, string(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return receipt, fmt.Errorf("deploy hook returned HTTP %d", resp.StatusCode)
	}

	// Verify the receipt is valid JSON (truthful receipt).
	var verify map[string]any
	if err := json.Unmarshal([]byte(receipt), &verify); err != nil {
		return receipt, fmt.Errorf("receipt is not valid JSON: %w", err)
	}

	return receipt, nil
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
