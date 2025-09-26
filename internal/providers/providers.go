package providers

import (
	"archive/zip"
	"bytes"
	"context"
	"docker-app/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DeploymentProvider interface for all deployment providers
type DeploymentProvider interface {
	Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error
	GetType() string
}

// S3Provider handles deployment to AWS S3
type S3Provider struct{}

type S3Config struct {
	Bucket          string `json:"bucket"`
	Key             string `json:"key"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func NewS3Provider() *S3Provider {
	return &S3Provider{}
}

func (p *S3Provider) GetType() string {
	return "s3"
}

func (p *S3Provider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config S3Config
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid S3 config: %v", err)
	}

	// TODO: Implement actual S3 upload when AWS SDK is added
	// For now, just log the deployment
	log.Printf("S3 DEPLOYMENT: Would upload %s to s3://%s/%s in region %s",
		artifactPath, config.Bucket, config.Key, config.Region)

	return nil
}

// EmailProvider handles deployment via email
type EmailProvider struct{}

type EmailConfig struct {
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

	// For now, just log the email deployment (you can integrate with actual email service)
	log.Printf("EMAIL DEPLOYMENT: Would send %s to %v with subject '%s'",
		artifactPath, config.To, config.Subject)

	// TODO: Integrate with actual email service (SendGrid, SES, etc.)
	return nil
}

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

// LocalProvider handles local file system deployment
type LocalProvider struct{}

type LocalConfig struct {
	Path string `json:"path"`
}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

func (p *LocalProvider) GetType() string {
	return "local"
}

func (p *LocalProvider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config LocalConfig
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid local config: %v", err)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(config.Path)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Handle different artifact types
	if strings.HasPrefix(artifactPath, "container:") {
		// Container artifact - create JSON info file
		return p.deployContainerInfo(runnable, deployment, artifactPath, config.Path)
	} else {
		// File artifact - copy the file
		return p.deployFile(artifactPath, config.Path)
	}
}

// deployFile handles regular file deployment
func (p *LocalProvider) deployFile(artifactPath, destPath string) error {
	// Copy file
	src, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	log.Printf("Successfully deployed file to local path: %s", destPath)
	return nil
}

// deployContainerInfo handles container deployment by creating a JSON info file
func (p *LocalProvider) deployContainerInfo(runnable models.Runnable, deployment models.Deployment, artifactPath, destPath string) error {
	// Parse container info from artifact path: "container:containerID:containerName"
	parts := strings.Split(artifactPath, ":")
	if len(parts) != 3 {
		return fmt.Errorf("invalid container artifact format: %s", artifactPath)
	}

	containerInfo := map[string]interface{}{
		"type":           "docker_container",
		"runnable_name":  runnable.Name,
		"runnable_type":  runnable.Type,
		"container_id":   parts[1],
		"container_name": parts[2],
		"deployment_id":  deployment.ID,
		"status":         "running",
		"artifact_path":  artifactPath,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(containerInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal container info: %v", err)
	}

	// Write to file
	err = os.WriteFile(destPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write container info file: %v", err)
	}

	log.Printf("Successfully deployed container info to local path: %s", destPath)
	return nil
}

// ProviderManager manages all deployment providers
type ProviderManager struct {
	providers map[string]DeploymentProvider
}

func NewProviderManager() *ProviderManager {
	pm := &ProviderManager{
		providers: make(map[string]DeploymentProvider),
	}

	// Register providers
	pm.RegisterProvider(NewS3Provider())
	pm.RegisterProvider(NewEmailProvider())
	pm.RegisterProvider(NewWebhookProvider())
	pm.RegisterProvider(NewLocalProvider())

	return pm
}

func (pm *ProviderManager) RegisterProvider(provider DeploymentProvider) {
	pm.providers[provider.GetType()] = provider
}

func (pm *ProviderManager) GetProvider(providerType string) (DeploymentProvider, error) {
	provider, exists := pm.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not found", providerType)
	}
	return provider, nil
}

// Artifact generation utilities
func CreateZipArchive(sourceDir, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Create zip entry
		writer, err := archive.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func SaveDockerImage(dockerClient *client.Client, imageID, imagePath string) error {
	ctx := context.Background()

	// Save image to tar
	imageReader, err := dockerClient.ImageSave(ctx, []string{imageID})
	if err != nil {
		return fmt.Errorf("failed to save Docker image: %v", err)
	}
	defer imageReader.Close()

	// Create output file
	outFile, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Copy image data to file
	_, err = io.Copy(outFile, imageReader)
	if err != nil {
		return fmt.Errorf("failed to write image to file: %v", err)
	}

	return nil
}

func BuildDockerImage(dockerClient *client.Client, buildContext io.Reader, dockerfile, tag string) (string, error) {
	ctx := context.Background()

	buildResponse, err := dockerClient.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfile,
		Remove:     true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to build Docker image: %v", err)
	}
	defer buildResponse.Body.Close()

	// Read build output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, buildResponse.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read build output: %v", err)
	}

	log.Printf("Docker build output: %s", buf.String())
	return tag, nil
}
