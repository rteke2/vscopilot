package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"vscopilot/internal/bridge"
	"vscopilot/internal/copilot"
)

type app struct {
	gasWebhookURL string
	triggerToken  string
}

func main() {
	app := app{
		gasWebhookURL: strings.TrimSpace(os.Getenv("GAS_WEBHOOK_URL")),
		triggerToken:  strings.TrimSpace(os.Getenv("TRIGGER_TOKEN")),
	}

	if app.gasWebhookURL == "" {
		log.Fatal("GAS_WEBHOOK_URL is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.health)
	mux.HandleFunc("/trigger", app.trigger)

	addr := envOr("LISTEN_ADDR", ":8080")
	log.Printf("listener started on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (a app) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a app) trigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if !a.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	vscodeRunning, processLine := detectVscodeProcess()

	snapshot, err := copilot.ReadLatestChat()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	hostname, _ := os.Hostname()
	payload := bridge.Payload{
		TriggeredAt:     time.Now().UTC(),
		Host:            hostname,
		VscodeRunning:   vscodeRunning,
		VscodeProcess:   processLine,
		SourceLogFile:   snapshot.LogFile,
		LatestUser:      snapshot.LatestUser,
		LatestAssistant: snapshot.LatestAssistant,
		RawExcerpt:      snapshot.RawExcerpt,
	}

	if err := postJSON(a.gasWebhookURL, payload); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("post to GAS failed: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "forwarded",
		"sourceLogFile": snapshot.LogFile,
	})
}

func (a app) authorized(r *http.Request) bool {
	if a.triggerToken == "" {
		return true
	}
	return strings.TrimSpace(r.Header.Get("X-Trigger-Token")) == a.triggerToken
}

func detectVscodeProcess() (bool, string) {
	cmd := exec.Command("bash", "-lc", "pgrep -af 'code|code-insiders|cursor' | head -n 1")
	out, err := cmd.Output()
	if err != nil {
		return false, ""
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false, ""
	}
	return true, line
}

func postJSON(url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New(res.Status)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
