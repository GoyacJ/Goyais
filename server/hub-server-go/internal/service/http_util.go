package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// postJSON is a simple helper for hub→worker HTTP calls.
func postJSON(url string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("worker returned %d for %s", resp.StatusCode, url)
	}
	return nil
}
