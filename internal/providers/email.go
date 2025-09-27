package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"docker-app/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// EmailProvider handles deployment via email
type EmailProvider struct{}

type EmailConfig struct {
	Transport string `json:"transport"` // "smtp", "ses", "http"

	// SMTP configuration
	SMTPHost string `json:"smtp_host,omitempty"`
	SMTPPort int    `json:"smtp_port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// AWS SES configuration
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`

	// HTTP API configuration
	APIURL  string            `json:"api_url,omitempty"`
	APIKey  string            `json:"api_key,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// Common fields
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

func NewEmailProvider() *EmailProvider {
	return &EmailProvider{}
}

func (p *EmailProvider) GetType() string {
	return "email"
}

func (p *EmailProvider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config EmailConfig
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid email config: %v", err)
	}

	// Check if artifact exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return fmt.Errorf("artifact file does not exist: %s", artifactPath)
	}

	// Route to appropriate transport method
	switch strings.ToLower(config.Transport) {
	case "smtp":
		return p.sendViaSMTP(ctx, config, artifactPath)
	case "ses":
		return p.sendViaSES(ctx, config, artifactPath)
	case "http":
		return p.sendViaHTTP(ctx, config, artifactPath)
	default:
		return fmt.Errorf("unsupported email transport: %s (supported: smtp, ses, http)", config.Transport)
	}
}

// sendViaSMTP sends email using SMTP
func (p *EmailProvider) sendViaSMTP(ctx context.Context, config EmailConfig, artifactPath string) error {
	// Compose email message
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n\r\nArtifact: %s",
		config.From, strings.Join(config.To, ","), config.Subject, config.Body, artifactPath)

	// Set up authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)

	// Send email
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort),
		auth,
		config.From,
		config.To,
		[]byte(message),
	)
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %v", err)
	}

	log.Printf("EMAIL DEPLOYMENT (SMTP): Successfully sent email to %v with subject '%s' for artifact %s",
		config.To, config.Subject, artifactPath)

	return nil
}

// sendViaSES sends email using AWS SES
func (p *EmailProvider) sendViaSES(ctx context.Context, config EmailConfig, artifactPath string) error {
	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(config.Region),
		awsconfig.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     config.AccessKeyID,
				SecretAccessKey: config.SecretAccessKey,
			}, nil
		}))),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config for SES: %v", err)
	}

	// Create SES v2 client
	sesClient := sesv2.NewFromConfig(awsCfg)

	// Send email
	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(config.From),
		Destination: &sesv2types.Destination{
			ToAddresses: config.To,
		},
		Content: &sesv2types.EmailContent{
			Simple: &sesv2types.Message{
				Subject: &sesv2types.Content{
					Data: aws.String(config.Subject),
				},
				Body: &sesv2types.Body{
					Text: &sesv2types.Content{
						Data: aws.String(fmt.Sprintf("%s\r\n\r\nArtifact: %s", config.Body, artifactPath)),
					},
				},
			},
		},
	}

	_, err = sesClient.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %v", err)
	}

	log.Printf("EMAIL DEPLOYMENT (SES): Successfully sent email to %v with subject '%s' for artifact %s",
		config.To, config.Subject, artifactPath)

	return nil
}

// sendViaHTTP sends email using HTTP API
func (p *EmailProvider) sendViaHTTP(ctx context.Context, config EmailConfig, artifactPath string) error {
	// Prepare request payload
	payload := map[string]interface{}{
		"from":    config.From,
		"to":      config.To,
		"subject": config.Subject,
		"body":    fmt.Sprintf("%s\r\n\r\nArtifact: %s", config.Body, artifactPath),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal HTTP payload: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", config.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP API returned status %d", resp.StatusCode)
	}

	log.Printf("EMAIL DEPLOYMENT (HTTP): Successfully sent email to %v with subject '%s' for artifact %s",
		config.To, config.Subject, artifactPath)

	return nil
}
