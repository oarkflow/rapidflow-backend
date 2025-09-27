package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"docker-app/internal/models"

	"golang.org/x/crypto/ssh"
)

// NginxProvider handles deployment to VPS with native Nginx (not Nginx Proxy Manager)
type NginxProvider struct{}

type NginxConfig struct {
	Host            string `json:"host"`              // VPS hostname/IP
	SSHUser         string `json:"ssh_user"`          // SSH username
	SSHKeyPath      string `json:"ssh_key_path"`      // Path to SSH private key
	SSHPort         string `json:"ssh_port"`          // SSH port (default: 22)
	DockerHost      string `json:"docker_host"`       // Docker daemon host (optional)
	Domain          string `json:"domain"`            // Domain name for the service
	ServicePort     string `json:"service_port"`      // Port the service runs on in container
	ContainerName   string `json:"container_name"`    // Name for the deployed container
	ImageName       string `json:"image_name"`        // Docker image to deploy
	NginxConfigPath string `json:"nginx_config_path"` // Path to Nginx sites-enabled directory (default: /etc/nginx/sites-enabled)
	NginxRestartCmd string `json:"nginx_restart_cmd"` // Command to restart Nginx (default: systemctl restart nginx)
	SSL             bool   `json:"ssl"`               // Enable SSL configuration
	SSLCertPath     string `json:"ssl_cert_path"`     // Path to SSL certificate
	SSLKeyPath      string `json:"ssl_key_path"`      // Path to SSL private key
}

func NewNginxProvider() *NginxProvider {
	return &NginxProvider{}
}

func (p *NginxProvider) GetType() string {
	return "nginx"
}

func (p *NginxProvider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var config NginxConfig
	if err := json.Unmarshal([]byte(deployment.Config), &config); err != nil {
		return fmt.Errorf("invalid Nginx config: %v", err)
	}

	// Set defaults
	if config.NginxConfigPath == "" {
		config.NginxConfigPath = "/etc/nginx/sites-enabled"
	}
	if config.NginxRestartCmd == "" {
		config.NginxRestartCmd = "systemctl restart nginx"
	}

	log.Printf("Starting Nginx deployment to %s for domain %s", config.Host, config.Domain)

	// Step 1: Deploy container to VPS
	if err := p.deployContainerToVPS(ctx, config, runnable, artifactPath); err != nil {
		return fmt.Errorf("failed to deploy container: %v", err)
	}

	// Step 2: Configure Nginx
	if err := p.configureNginx(ctx, config); err != nil {
		return fmt.Errorf("failed to configure Nginx: %v", err)
	}

	log.Printf("Successfully deployed to VPS and configured Nginx for %s", config.Domain)
	return nil
}

// SSH helper methods for NginxProvider
func (p *NginxProvider) connectSSH(host, user, keyPath, sshPort string) (*ssh.Client, error) {
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

func (p *NginxProvider) runSSHCommand(client *ssh.Client, command string) error {
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

func (p *NginxProvider) uploadFileViaSSH(client *ssh.Client, content, remotePath string) error {
	// Create session for file upload
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Use SCP-like approach with cat
	cmd := fmt.Sprintf("cat > %s", remotePath)
	session.Stdin = strings.NewReader(content)

	// Run command
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v, output: %s", err, string(output))
	}

	log.Printf("Successfully uploaded file to %s", remotePath)
	return nil
}

func (p *NginxProvider) deployContainerToVPS(ctx context.Context, config NginxConfig, runnable models.Runnable, artifactPath string) error {
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

func (p *NginxProvider) configureNginx(ctx context.Context, config NginxConfig) error {
	// Establish SSH connection
	client, err := p.connectSSH(config.Host, config.SSHUser, config.SSHKeyPath, config.SSHPort)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS for Nginx config: %v", err)
	}
	defer client.Close()

	// Generate Nginx configuration
	nginxConfig := p.generateNginxConfig(config)

	// Create temporary config file path
	configFileName := fmt.Sprintf("%s.conf", config.Domain)
	configFilePath := fmt.Sprintf("/tmp/%s", configFileName)

	// Upload config file via SSH
	if err := p.uploadFileViaSSH(client, nginxConfig, configFilePath); err != nil {
		return fmt.Errorf("failed to upload Nginx config: %v", err)
	}

	// Move config to proper location
	targetPath := fmt.Sprintf("%s/%s", config.NginxConfigPath, configFileName)
	commands := []string{
		fmt.Sprintf("sudo mv %s %s", configFilePath, targetPath),
		fmt.Sprintf("sudo chown root:root %s", targetPath),
		fmt.Sprintf("sudo chmod 644 %s", targetPath),
		"sudo nginx -t",        // Test configuration
		config.NginxRestartCmd, // Restart Nginx
	}

	// Execute commands
	for _, cmd := range commands {
		if err := p.runSSHCommand(client, cmd); err != nil {
			return fmt.Errorf("failed to execute command '%s': %v", cmd, err)
		}
	}

	log.Printf("Successfully configured Nginx for domain %s on VPS %s", config.Domain, config.Host)
	return nil
}

func (p *NginxProvider) generateNginxConfig(config NginxConfig) string {
	var nginxConfig string

	if config.SSL {
		nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name %s;

    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://127.0.0.1:%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`, config.Domain, config.Domain, config.SSLCertPath, config.SSLKeyPath, config.ServicePort)
	} else {
		nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://127.0.0.1:%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`, config.Domain, config.ServicePort)
	}

	return nginxConfig
}
