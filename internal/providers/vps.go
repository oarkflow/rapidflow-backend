package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"docker-app/internal/models"

	"golang.org/x/crypto/ssh"
)

// VPSProvider handles deployment to a remote VPS with Docker and Nginx Proxy Manager
type VPSProvider struct{}

type VPSConfig struct {
	Host          string `json:"host"`           // VPS hostname/IP
	SSHUser       string `json:"ssh_user"`       // SSH username
	SSHKeyPath    string `json:"ssh_key_path"`   // Path to SSH private key
	SSHPort       string `json:"ssh_port"`       // SSH port (default: 22)
	DockerHost    string `json:"docker_host"`    // Docker daemon host (optional, defaults to local)
	NginxPMURL    string `json:"nginx_pm_url"`   // Nginx Proxy Manager URL
	NginxPMUser   string `json:"nginx_pm_user"`  // Nginx Proxy Manager username
	NginxPMPass   string `json:"nginx_pm_pass"`  // Nginx Proxy Manager password
	Domain        string `json:"domain"`         // Domain name for the service
	ServicePort   string `json:"service_port"`   // Port the service runs on in container
	ContainerName string `json:"container_name"` // Name for the deployed container
	ImageName     string `json:"image_name"`     // Docker image to deploy
}

func NewVPSProvider() *VPSProvider {
	return &VPSProvider{}
}

func (p *VPSProvider) GetType() string {
	return "vps"
}

func (p *VPSProvider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config VPSConfig
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid VPS config: %v", err)
	}

	log.Printf("Starting VPS deployment to %s for domain %s", config.Host, config.Domain)

	// Step 1: Deploy container to VPS
	if err := p.deployContainerToVPS(ctx, config, runnable, artifactPath); err != nil {
		return fmt.Errorf("failed to deploy container: %v", err)
	}

	// Step 2: Configure Nginx Proxy Manager
	if err := p.configureNginxProxyManager(ctx, config); err != nil {
		return fmt.Errorf("failed to configure Nginx Proxy Manager: %v", err)
	}

	log.Printf("Successfully deployed to VPS and configured proxy for %s", config.Domain)
	return nil
}

// SSH helper methods for VPSProvider
func (p *VPSProvider) connectSSH(host, user, keyPath, sshPort string) (*ssh.Client, error) {
	// Read private key
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %v", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %v", err)
	}

	// SSH client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
	}

	// Use custom port if provided, otherwise default to 22
	port := sshPort
	if port == "" {
		port = "22"
	}

	// Connect to SSH server
	client, err := ssh.Dial("tcp", host+":"+port, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH: %v", err)
	}

	return client, nil
}

func (p *VPSProvider) runSSHCommand(client *ssh.Client, command string) error {
	// Create session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Run command
	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Errorf("command failed: %v, output: %s", err, string(output))
	}

	log.Printf("SSH command output: %s", string(output))
	return nil
}

func (p *VPSProvider) deployContainerToVPS(ctx context.Context, config VPSConfig, runnable models.Runnable, artifactPath string) error {
	// Establish SSH connection
	client, err := p.connectSSH(config.Host, config.SSHUser, config.SSHKeyPath, config.SSHPort)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer client.Close()

	// Commands to run on the VPS
	commands := []string{
		fmt.Sprintf("docker pull %s", config.ImageName),
		fmt.Sprintf("docker stop %s || true", config.ContainerName),
		fmt.Sprintf("docker rm %s || true", config.ContainerName),
		fmt.Sprintf("docker run -d --name %s -p %s:%s %s",
			config.ContainerName, config.ServicePort, config.ServicePort, config.ImageName),
		fmt.Sprintf("docker ps | grep %s", config.ContainerName),
	}

	// Execute commands
	for _, cmd := range commands {
		if err := p.runSSHCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to execute command '%s': %v", cmd, err)
		}
	}

	log.Printf("Successfully deployed container %s to VPS %s", config.ContainerName, config.Host)
	return nil
}

func (p *VPSProvider) configureNginxProxyManager(ctx context.Context, config VPSConfig) error {
	// Nginx Proxy Manager API endpoints
	loginURL := fmt.Sprintf("%s/api/tokens", config.NginxPMURL)
	hostsURL := fmt.Sprintf("%s/api/nginx/proxy-hosts", config.NginxPMURL)

	// Step 1: Authenticate and get token
	token, err := p.authenticateWithNginxPM(ctx, loginURL, config.NginxPMUser, config.NginxPMPass)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Nginx Proxy Manager: %v", err)
	}

	// Step 2: Create proxy host
	if err := p.createProxyHost(ctx, hostsURL, token, config); err != nil {
		return fmt.Errorf("failed to create proxy host: %v", err)
	}

	log.Printf("Successfully configured Nginx Proxy Manager for domain %s", config.Domain)
	return nil
}

func (p *VPSProvider) authenticateWithNginxPM(ctx context.Context, loginURL, username, password string) (string, error) {
	authPayload := map[string]string{
		"identity": username,
		"secret":   password,
	}

	jsonData, err := json.Marshal(authPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var authResponse struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return "", err
	}

	return authResponse.Token, nil
}

func (p *VPSProvider) createProxyHost(ctx context.Context, hostsURL, token string, config VPSConfig) error {
	// Nginx Proxy Manager proxy host configuration
	hostConfig := map[string]interface{}{
		"domain_names": []string{config.Domain},
		"forward_host": "127.0.0.1", // Assuming container is accessible locally
		"forward_port": config.ServicePort,
		"ssl_enabled":  true,
		"ssl_email":    config.NginxPMUser,
		"ssl_force":    true,
		"enabled":      true,
	}

	jsonData, err := json.Marshal(hostConfig)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hostsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to create proxy host, status %d", resp.StatusCode)
	}

	log.Printf("Created proxy host for %s forwarding to port %s", config.Domain, config.ServicePort)
	return nil
}
