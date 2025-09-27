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
	"net/smtp"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"
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
	var s3Config S3Config
	if err := json.Unmarshal([]byte(deployment.Config), &s3Config); err != nil {
		return fmt.Errorf("invalid S3 config: %v", err)
	}

	// Check if artifact exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return fmt.Errorf("artifact file does not exist: %s", artifactPath)
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Config.Region),
		awsconfig.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     s3Config.AccessKeyID,
				SecretAccessKey: s3Config.SecretAccessKey,
			}, nil
		}))),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	// Open artifact file
	file, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact file: %v", err)
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Upload to S3
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s3Config.Bucket),
		Key:           aws.String(s3Config.Key),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	log.Printf("Successfully uploaded %s to s3://%s/%s", artifactPath, s3Config.Bucket, s3Config.Key)
	return nil
}

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
	pm.RegisterProvider(NewVPSProvider())
	pm.RegisterProvider(NewNginxProvider())

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
