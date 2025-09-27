package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"docker-app/internal/models"
)

// WebhookProvider handles deployment via webhook
type WebhookProvider struct{}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

func NewWebhookProvider() *WebhookProvider {
	return &WebhookProvider{}
}

func (p *WebhookProvider) GetType() string {
	return "webhook"
}

func (p *WebhookProvider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config WebhookConfig
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid webhook config: %v", err)
	}

	// Read artifact file
	file, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact: %v", err)
	}
	defer file.Close()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, config.Method, config.URL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Successfully sent webhook to %s", config.URL)
	return nil
}
