package providers

import (
	"archive/zip"
	"bytes"
	"context"
	"docker-app/internal/models"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Provider interface defines the contract for all deployment providers.
// This interface enables an extensible plugin system where new deployment
// providers can be added without modifying existing code.
//
// To create a new provider:
// 1. Implement the Provider interface
// 2. Register it with the ProviderManager using RegisterProvider()
type Provider interface {
	// Deploy executes the deployment logic for a specific provider
	Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error

	// GetType returns the unique identifier for this provider type
	GetType() string
}

// EmailProvider handles deployment via email
// WebhookProvider handles deployment via webhook

// ProviderManager manages all deployment providers and provides
// a registry for extensible provider implementations.
//
// Usage:
//
//	pm := NewProviderManager()
//	pm.RegisterProvider(NewMyCustomProvider())
//	provider, err := pm.GetProvider("my-custom-type")
type ProviderManager struct {
	providers map[string]Provider
}

func NewProviderManager() *ProviderManager {
	pm := &ProviderManager{
		providers: make(map[string]Provider),
	}

	// Register providers
	pm.RegisterProvider(NewS3Provider())
	pm.RegisterProvider(NewEmailProvider())
	pm.RegisterProvider(NewWebhookProvider())
	pm.RegisterProvider(NewLocalProvider())
	pm.RegisterProvider(NewVPSProvider())
	pm.RegisterProvider(NewNginxProvider())

	return pm
}

// RegisterProvider adds a new provider to the registry.
// The provider's type (returned by GetType()) will be used as the key.
func (pm *ProviderManager) RegisterProvider(provider Provider) {
	pm.providers[provider.GetType()] = provider
}

// GetProvider retrieves a provider by its type.
// Returns an error if the provider type is not registered.
func (pm *ProviderManager) GetProvider(providerType string) (Provider, error) {
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

	buildResponse, err := dockerClient.ImageBuild(ctx, buildContext, dockertypes.ImageBuildOptions{
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
