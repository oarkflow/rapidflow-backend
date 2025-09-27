package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"docker-app/internal/models"
)

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
