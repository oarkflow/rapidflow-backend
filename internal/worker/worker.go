package worker

import (
	"bufio"
	"bytes"
	"context"
	"docker-app/internal/models"
	"docker-app/internal/providers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jmoiron/sqlx"
)

type Worker struct {
	DB              *sqlx.DB
	Docker          *client.Client
	runningJobs     map[int]context.CancelFunc
	mutex           sync.RWMutex
	providerManager *providers.ProviderManager
}

func NewWorker(db *sqlx.DB) (*Worker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Worker{
		DB:              db,
		Docker:          cli,
		runningJobs:     make(map[int]context.CancelFunc),
		providerManager: providers.NewProviderManager(),
	}, nil
}

// addRunningJob adds a job to the running jobs map with its cancel function
func (w *Worker) addRunningJob(jobID int, cancel context.CancelFunc) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.runningJobs[jobID] = cancel
}

// removeRunningJob removes a job from the running jobs map
func (w *Worker) removeRunningJob(jobID int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	delete(w.runningJobs, jobID)
}

// CancelJob cancels a running job by its ID
func (w *Worker) CancelJob(jobID int) error {
	w.mutex.RLock()
	cancel, exists := w.runningJobs[jobID]
	w.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("job %d is not currently running", jobID)
	}

	cancel()
	return nil
}

func getBaseImage(language, version string) string {
	switch language {
	case "golang", "go":
		if version != "" {
			return fmt.Sprintf("golang:%s", version)
		}
		return "golang:latest"
	case "python", "python3":
		if version != "" {
			return fmt.Sprintf("python:%s", version)
		}
		return "python:latest"
	case "node", "javascript":
		if version != "" {
			return fmt.Sprintf("node:%s", version)
		}
		return "node:latest"
	case "scala":
		if version != "" {
			return fmt.Sprintf("hseeberger/scala-sbt:%s", version)
		}
		return "hseeberger/scala-sbt:latest"
	default:
		return "ubuntu:latest"
	}
}

// LanguageInfo holds detected language and version information
type LanguageInfo struct {
	Language string
	Version  string
}

// detectLanguageAndVersion automatically detects language and version from the project folder
func detectLanguageAndVersion(projectPath string) (*LanguageInfo, error) {
	log.Printf("Detecting language and version in: %s", projectPath)

	// Check for Go
	if goInfo := detectGo(projectPath); goInfo != nil {
		return goInfo, nil
	}

	// Check for Node.js
	if nodeInfo := detectNode(projectPath); nodeInfo != nil {
		return nodeInfo, nil
	}

	// Check for Python
	if pythonInfo := detectPython(projectPath); pythonInfo != nil {
		return pythonInfo, nil
	}

	// Check for Java/Scala
	if javaInfo := detectJavaScala(projectPath); javaInfo != nil {
		return javaInfo, nil
	}

	// Default to Go if nothing detected
	log.Printf("No specific language detected, defaulting to golang")
	return &LanguageInfo{Language: "golang", Version: "latest"}, nil
}

// detectGo detects Go projects and version
func detectGo(projectPath string) *LanguageInfo {
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		log.Printf("Detected Go project (go.mod found)")

		// Try to extract Go version from go.mod
		content, err := os.ReadFile(goModPath)
		if err != nil {
			return &LanguageInfo{Language: "golang", Version: "latest"}
		}

		// Look for "go 1.21" pattern
		goVersionRegex := regexp.MustCompile(`go\s+(\d+\.\d+(?:\.\d+)?)`)
		matches := goVersionRegex.FindStringSubmatch(string(content))
		if len(matches) > 1 {
			return &LanguageInfo{Language: "golang", Version: matches[1]}
		}

		return &LanguageInfo{Language: "golang", Version: "latest"}
	}

	// Check for .go files
	matches, _ := filepath.Glob(filepath.Join(projectPath, "*.go"))
	if len(matches) > 0 {
		log.Printf("Detected Go project (.go files found)")
		return &LanguageInfo{Language: "golang", Version: "latest"}
	}

	return nil
}

// detectNode detects Node.js projects and version
func detectNode(projectPath string) *LanguageInfo {
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(packageJsonPath); err == nil {
		log.Printf("Detected Node.js project (package.json found)")

		// Try to extract Node version from package.json
		content, err := os.ReadFile(packageJsonPath)
		if err != nil {
			return &LanguageInfo{Language: "node", Version: "latest"}
		}

		var packageData map[string]interface{}
		if err := json.Unmarshal(content, &packageData); err == nil {
			if engines, ok := packageData["engines"].(map[string]interface{}); ok {
				if nodeVersion, ok := engines["node"].(string); ok {
					// Clean up version string (remove ^ ~ >= etc)
					cleanVersion := regexp.MustCompile(`[^\d\.]`).ReplaceAllString(nodeVersion, "")
					if cleanVersion != "" {
						return &LanguageInfo{Language: "node", Version: cleanVersion}
					}
				}
			}
		}

		return &LanguageInfo{Language: "node", Version: "latest"}
	}

	// Check for .js files
	matches, _ := filepath.Glob(filepath.Join(projectPath, "*.js"))
	if len(matches) > 0 {
		log.Printf("Detected Node.js project (.js files found)")
		return &LanguageInfo{Language: "node", Version: "latest"}
	}

	return nil
}

// detectPython detects Python projects and version
func detectPython(projectPath string) *LanguageInfo {
	// Check for requirements.txt
	reqPath := filepath.Join(projectPath, "requirements.txt")
	if _, err := os.Stat(reqPath); err == nil {
		log.Printf("Detected Python project (requirements.txt found)")
		return &LanguageInfo{Language: "python", Version: "latest"}
	}

	// Check for setup.py
	setupPath := filepath.Join(projectPath, "setup.py")
	if _, err := os.Stat(setupPath); err == nil {
		log.Printf("Detected Python project (setup.py found)")
		return &LanguageInfo{Language: "python", Version: "latest"}
	}

	// Check for pyproject.toml
	pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		log.Printf("Detected Python project (pyproject.toml found)")
		return &LanguageInfo{Language: "python", Version: "latest"}
	}

	// Check for .py files
	matches, _ := filepath.Glob(filepath.Join(projectPath, "*.py"))
	if len(matches) > 0 {
		log.Printf("Detected Python project (.py files found)")
		return &LanguageInfo{Language: "python", Version: "latest"}
	}

	return nil
}

// detectJavaScala detects Java/Scala projects and version
func detectJavaScala(projectPath string) *LanguageInfo {
	// Check for build.sbt (Scala)
	sbtPath := filepath.Join(projectPath, "build.sbt")
	if _, err := os.Stat(sbtPath); err == nil {
		log.Printf("Detected Scala project (build.sbt found)")
		return &LanguageInfo{Language: "scala", Version: "latest"}
	}

	// Check for pom.xml (Maven - Java)
	pomPath := filepath.Join(projectPath, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		log.Printf("Detected Java project (pom.xml found)")
		return &LanguageInfo{Language: "java", Version: "latest"}
	}

	// Check for build.gradle (Gradle - Java)
	gradlePath := filepath.Join(projectPath, "build.gradle")
	if _, err := os.Stat(gradlePath); err == nil {
		log.Printf("Detected Java project (build.gradle found)")
		return &LanguageInfo{Language: "java", Version: "latest"}
	}

	return nil
}

// cloneRepository clones a git repository to a temporary directory
func cloneRepository(repoURL, branch, targetDir string) error {
	log.Printf("Cloning repository %s (branch: %s) to %s", repoURL, branch, targetDir)

	var cmd *exec.Cmd
	if branch != "" {
		cmd = exec.Command("git", "clone", "--branch", branch, "--depth", "1", repoURL, targetDir)
	} else {
		cmd = exec.Command("git", "clone", "--depth", "1", repoURL, targetDir)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %s, output: %s", err, string(output))
	}

	log.Printf("Repository cloned successfully")
	return nil
}

// cleanupTemporaryResources cleans up temporary containers, images, and directories
func (w *Worker) cleanupTemporaryResources(jobID int, containerID, tempDir string) {
	log.Printf("Cleaning up temporary resources for job %d", jobID)

	ctx := context.Background()

	// Remove container if it exists
	if containerID != "" {
		log.Printf("Removing temporary container: %s", containerID)
		err := w.Docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			log.Printf("Failed to remove container %s: %v", containerID, err)
		}
	}

	// Remove temporary directory
	if tempDir != "" {
		log.Printf("Removing temporary directory: %s", tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			log.Printf("Failed to remove temporary directory %s: %v", tempDir, err)
		}
	}

	log.Printf("Cleanup completed for job %d", jobID)
}

// RemoveContainerByName removes a container by its name
func (w *Worker) RemoveContainerByName(containerName string) error {
	ctx := context.Background()

	// Find container by name
	containers, err := w.Docker.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	for _, container := range containers {
		for _, name := range container.Names {
			// Docker prefixes container names with "/"
			if strings.TrimPrefix(name, "/") == containerName {
				log.Printf("Removing container: %s (ID: %s)", containerName, container.ID)
				err := w.Docker.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})
				if err != nil {
					return fmt.Errorf("failed to remove container %s: %v", containerName, err)
				}
				return nil
			}
		}
	}

	log.Printf("Container %s not found", containerName)
	return nil
}

// CleanupJobResources cleans up all resources associated with a job
func (w *Worker) CleanupJobResources(jobID int, containerID, tempDir string) {
	log.Printf("Cleaning up job %d resources", jobID)

	ctx := context.Background()

	// Remove main container if it exists
	if containerID != "" {
		log.Printf("Removing job container: %s", containerID)
		err := w.Docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			log.Printf("Failed to remove job container %s: %v", containerID, err)
		}
	}

	// Remove temporary directory
	if tempDir != "" {
		log.Printf("Removing temporary directory: %s", tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			log.Printf("Failed to remove temporary directory %s: %v", tempDir, err)
		}
	}
}

func (w *Worker) RunJob(jobID int) error {
	return w.RunJobWithContext(context.Background(), jobID)
}

func (w *Worker) RunJobWithContext(ctx context.Context, jobID int) error {
	// Create a cancellable context for this job
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register this job as running
	w.addRunningJob(jobID, cancel)
	defer w.removeRunningJob(jobID)

	log.Printf("Starting job %d", jobID)

	// Check if job was cancelled before we start
	var job models.Job
	err := w.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", jobID)
	if err != nil {
		return err
	}

	if job.Cancelled {
		log.Printf("Job %d was cancelled before starting", jobID)
		return nil
	}

	// Update status to running
	_, err = w.DB.Exec("UPDATE jobs SET status = 'running', started_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
	if err != nil {
		return err
	}

	// Handle repository cloning and language detection
	var projectPath string
	var tempDir string
	var isTemporary bool

	if job.Temporary != nil && *job.Temporary {
		isTemporary = true
	}

	// If repo URL is provided, clone the repository
	if job.RepoURL != nil && *job.RepoURL != "" {
		// Create temporary directory for cloning
		tempDir = filepath.Join(os.TempDir(), fmt.Sprintf("rapidflow-repo-%d", jobID))
		err = os.MkdirAll(tempDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %v", err)
		}

		// Store temp directory path in database for cleanup
		if isTemporary {
			_, err = w.DB.Exec("UPDATE jobs SET temp_dir = ? WHERE id = ?", tempDir, jobID)
			if err != nil {
				log.Printf("Warning: failed to store temp directory path: %v", err)
			}
		}

		branch := "main" // default branch
		if job.Branch != nil && *job.Branch != "" {
			branch = *job.Branch
		}

		// Clone repository
		err = cloneRepository(*job.RepoURL, branch, tempDir)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %v", err)
		}

		// If folder is specified, use it as subdirectory
		if job.Folder != nil && *job.Folder != "" {
			projectPath = filepath.Join(tempDir, *job.Folder)
		} else {
			projectPath = tempDir
		}
	} else if job.Folder != nil && *job.Folder != "" {
		// Use local folder
		projectPath = *job.Folder
	} else {
		return fmt.Errorf("either repo_url or folder must be specified")
	}

	// Auto-detect language and version if not specified
	var detectedLanguage, detectedVersion string
	if job.Language == nil || *job.Language == "" || job.Version == nil || *job.Version == "" {
		log.Printf("Auto-detecting language and version for job %d", jobID)
		langInfo, err := detectLanguageAndVersion(projectPath)
		if err != nil {
			log.Printf("Language detection failed, using defaults: %v", err)
			detectedLanguage = "golang"
			detectedVersion = "latest"
		} else {
			detectedLanguage = langInfo.Language
			detectedVersion = langInfo.Version
		}

		// Update job with detected values if they weren't provided
		if job.Language == nil || *job.Language == "" {
			_, err = w.DB.Exec("UPDATE jobs SET language = ? WHERE id = ?", detectedLanguage, jobID)
			if err != nil {
				return err
			}
			job.Language = &detectedLanguage
		}

		if job.Version == nil || *job.Version == "" {
			_, err = w.DB.Exec("UPDATE jobs SET version = ? WHERE id = ?", detectedVersion, jobID)
			if err != nil {
				return err
			}
			job.Version = &detectedVersion
		}

		log.Printf("Job %d: Language=%s, Version=%s", jobID, *job.Language, *job.Version)
	}
	// Get env
	var envs []models.Environment
	err = w.DB.Select(&envs, "SELECT * FROM environments WHERE job_id = ?", jobID)
	if err != nil {
		return err
	}
	var envVars []string
	for _, env := range envs {
		envVars = append(envVars, fmt.Sprintf("%s=%s", env.Key, env.Value))
	}
	// Add branch if set
	if job.Branch != nil {
		envVars = append(envVars, fmt.Sprintf("BRANCH=%s", *job.Branch))
	}

	// Check for cancellation
	select {
	case <-jobCtx.Done():
		w.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
		return fmt.Errorf("job %d was cancelled", jobID)
	default:
	}

	// Setup ports
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	if job.ExposePorts != nil && *job.ExposePorts {
		for _, env := range envs {
			if env.Key == "PORT" {
				port := nat.Port(env.Value + "/tcp")
				exposedPorts[port] = struct{}{}
				portBindings[port] = []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: env.Value}}
			}
		}
	}
	// Setup base image
	versionStr := ""
	if job.Version != nil {
		versionStr = *job.Version
	}
	baseImage := getBaseImage(*job.Language, versionStr)
	fallback := false
	// Pull image
	log.Printf("Pulling image %s", baseImage)
	out, err := w.Docker.ImagePull(jobCtx, baseImage, types.ImagePullOptions{})
	if err != nil {
		log.Printf("Failed to pull image %s: %v, falling back to ubuntu", baseImage, err)
		fallback = true
		baseImage = "ubuntu:latest"
		out, err = w.Docker.ImagePull(jobCtx, baseImage, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(io.Discard, out)
		if err != nil {
			return err
		}
	} else {
		defer out.Close()
		_, err = io.Copy(io.Discard, out)
		if err != nil {
			return err
		}
	}
	log.Printf("Image pulled successfully")

	// Check for cancellation again
	select {
	case <-jobCtx.Done():
		w.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
		return fmt.Errorf("job %d was cancelled", jobID)
	default:
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	// Use the determined project path for volume binding
	if projectPath != "" {
		absPath, err := filepath.Abs(projectPath)
		if err != nil {
			return err
		}
		hostConfig.Binds = []string{fmt.Sprintf("%s:/workspace", absPath)}
	}

	resp, err := w.Docker.ContainerCreate(jobCtx, &container.Config{
		Image:        baseImage,
		Env:          envVars,
		Cmd:          []string{"sleep", "infinity"},
		Tty:          true,
		ExposedPorts: exposedPorts,
	}, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}
	containerID := resp.ID

	// Store container ID in database for potential cleanup
	_, err = w.DB.Exec("UPDATE jobs SET container_id = ? WHERE id = ?", containerID, jobID)
	if err != nil {
		return err
	}

	// Note: For temporary jobs, cleanup will be handled by stop-pipeline command
	// This allows users to access the server before manually stopping it
	if isTemporary {
		log.Printf("Job %d marked as temporary - resources will remain until pipeline is stopped", jobID)
	} else {
		// Only auto-cleanup non-temporary jobs
		defer w.Docker.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{Force: true})
	}

	// Start container
	err = w.Docker.ContainerStart(jobCtx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	log.Printf("Container started: %s", containerID)
	// Install language if fallback
	if fallback {
		scriptPath := fmt.Sprintf("scripts/%s-%s.sh", *job.Language, versionStr)
		if _, err := os.Stat(scriptPath); err == nil {
			log.Printf("Running install script %s", scriptPath)
			scriptContent, err := os.ReadFile(scriptPath)
			if err != nil {
				return err
			}
			// Create script in container
			execResp, err := w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("echo '%s' > /tmp/install.sh && chmod +x /tmp/install.sh", string(scriptContent))},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(jobCtx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			inspect, err := w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
			if err != nil {
				return err
			}
			if inspect.ExitCode != 0 {
				return fmt.Errorf("failed to create install script")
			}
			// Run script
			execResp, err = w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
				Cmd:          []string{"/tmp/install.sh"},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			hijacked, err := w.Docker.ContainerExecAttach(jobCtx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			defer hijacked.Close()
			var output bytes.Buffer
			scanner := bufio.NewScanner(hijacked.Reader)
			for scanner.Scan() {
				line := scanner.Text()
				log.Println(line)
				output.WriteString(line + "\n")

				// Check for cancellation while reading output
				select {
				case <-jobCtx.Done():
					w.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
					return fmt.Errorf("job %d was cancelled", jobID)
				default:
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			inspect, err = w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
			if err != nil {
				return err
			}
			if inspect.ExitCode != 0 {
				return fmt.Errorf("install script failed")
			}
		} else {
			log.Printf("No install script found for %s-%s", *job.Language, versionStr)
		}
	}
	if job.RepoName != nil {
		// Clone repo
		log.Printf("Cloning repo %s", *job.RepoName)
		execResp, err := w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
			Cmd:          []string{"sh", "-c", fmt.Sprintf("git clone %s /workspace", *job.RepoName)},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return err
		}
		err = w.Docker.ContainerExecStart(jobCtx, execResp.ID, types.ExecStartCheck{})
		if err != nil {
			return err
		}
		inspect, err := w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
		if err != nil {
			return err
		}
		if inspect.ExitCode != 0 {
			return fmt.Errorf("failed to clone repo")
		}
		log.Printf("Repo cloned")
		// Checkout branch if specified
		if job.Branch != nil && *job.Branch != "" {
			log.Printf("Checking out branch %s", *job.Branch)
			execResp, err := w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("cd /workspace && git checkout %s", *job.Branch)},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(jobCtx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			inspect, err := w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
			if err != nil {
				return err
			}
			if inspect.ExitCode != 0 {
				return fmt.Errorf("failed to checkout branch")
			}
			log.Printf("Branch checked out")
		}
	} else {
		log.Printf("Using local folder")
	}

	// Now run steps
	// Get steps
	var steps []models.Step
	err = w.DB.Select(&steps, "SELECT * FROM steps WHERE job_id = ? ORDER BY order_num", jobID)
	if err != nil {
		return err
	}
	log.Printf("Running %d steps", len(steps))
	for _, step := range steps {
		// Check for cancellation before each step
		select {
		case <-jobCtx.Done():
			w.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
			w.DB.Exec("UPDATE steps SET status = 'cancelled' WHERE job_id = ? AND status IN ('pending', 'running')", jobID)
			return fmt.Errorf("job %d was cancelled", jobID)
		default:
		}

		log.Printf("Running step %d", step.ID)
		// Update step status
		_, err = w.DB.Exec("UPDATE steps SET status = 'running' WHERE id = ?", step.ID)
		if err != nil {
			log.Printf("Error updating step status: %v", err)
		}
		// Get files for step
		var files []models.File
		err = w.DB.Select(&files, "SELECT * FROM files WHERE step_id = ?", step.ID)
		if err != nil {
			return err
		}
		// Create files
		for _, f := range files {
			content := f.Content
			// Exec to create file
			execResp, err := w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("echo '%s' > %s", content, f.Name)},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(jobCtx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			// Wait for exec
			inspect, err := w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
			if err != nil {
				return err
			}
			if inspect.ExitCode != 0 {
				output := "Failed to create file"
				w.DB.Exec("UPDATE steps SET status = 'failed', output = ? WHERE id = ?", output, step.ID)
				continue
			}
		}
		// Run the step content as bash
		if step.Type == "bash" {
			execResp, err := w.Docker.ContainerExecCreate(jobCtx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", step.Content},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			hijacked, err := w.Docker.ContainerExecAttach(jobCtx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			defer hijacked.Close()
			var output bytes.Buffer
			scanner := bufio.NewScanner(hijacked.Reader)
			for scanner.Scan() {
				line := scanner.Text()
				log.Println(line)
				output.WriteString(line + "\n")

				// Check for cancellation while reading step output
				select {
				case <-jobCtx.Done():
					w.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
					w.DB.Exec("UPDATE steps SET status = 'cancelled' WHERE job_id = ? AND status IN ('pending', 'running')", jobID)
					return fmt.Errorf("job %d was cancelled", jobID)
				default:
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			// Wait for exec
			inspect, err := w.Docker.ContainerExecInspect(jobCtx, execResp.ID)
			if err != nil {
				return err
			}
			status := "success"
			if inspect.ExitCode != 0 {
				status = "failed"
			}
			_, err = w.DB.Exec("UPDATE steps SET status = ?, output = ? WHERE id = ?", status, output.String(), step.ID)
			if err != nil {
				log.Printf("Error updating step: %v", err)
			}

			// If step failed, mark job as failed and stop
			if status == "failed" {
				w.DB.Exec("UPDATE jobs SET status = 'failed', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
				return fmt.Errorf("step %d failed", step.ID)
			}
		}
	}
	// Update job status to success
	_, err = w.DB.Exec("UPDATE jobs SET status = 'success', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
	if err != nil {
		return err
	}

	// Process runnables after successful build
	err = w.processRunnables(jobCtx, jobID, containerID, job)
	if err != nil {
		log.Printf("Error processing runnables for job %d: %v", jobID, err)
		// Don't fail the job if runnables fail, just log the error
	}

	return nil
}

// processRunnables handles the deployment/packaging phase after successful build
func (w *Worker) processRunnables(ctx context.Context, jobID int, containerID string, job models.Job) error {
	// Get runnables for this job
	var runnables []models.Runnable
	err := w.DB.Select(&runnables, "SELECT * FROM runnables WHERE job_id = ? AND status = 'pending'", jobID)
	if err != nil {
		return err
	}

	if len(runnables) == 0 {
		log.Printf("No runnables defined for job %d", jobID)
		return nil
	}

	log.Printf("Processing %d runnables for job %d", len(runnables), jobID)

	// Create temp directory for artifacts
	tempDir := fmt.Sprintf("/tmp/rapidflow-job-%d", jobID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Process each runnable
	for _, runnable := range runnables {
		err = w.processRunnable(ctx, runnable, containerID, tempDir, job)
		if err != nil {
			log.Printf("Failed to process runnable %s: %v", runnable.Name, err)
			w.DB.Exec("UPDATE runnables SET status = 'failed', output = ? WHERE id = ?", err.Error(), runnable.ID)
			continue
		}
	}

	return nil
}

// processRunnable processes a single runnable
func (w *Worker) processRunnable(ctx context.Context, runnable models.Runnable, containerID, tempDir string, job models.Job) error {
	log.Printf("Processing runnable: %s (type: %s)", runnable.Name, runnable.Type)

	// Update runnable status to running
	_, err := w.DB.Exec("UPDATE runnables SET status = 'running' WHERE id = ?", runnable.ID)
	if err != nil {
		return err
	}

	var artifactPath string

	// Parse runnable config
	var config models.RunnableConfig
	if err := json.Unmarshal([]byte(runnable.Config), &config); err != nil {
		return fmt.Errorf("failed to parse runnable config: %v", err)
	}

	switch runnable.Type {
	case "docker_container":
		artifactPath, err = w.handleDockerContainer(ctx, runnable, config, containerID, tempDir, job)
	case "docker_image":
		artifactPath, err = w.handleDockerImage(ctx, runnable, config, containerID, tempDir)
	case "artifacts":
		artifactPath, err = w.handleArtifacts(ctx, runnable, config, containerID, tempDir)
	case "serverless":
		artifactPath, err = w.handleServerless(ctx, runnable, config, containerID, tempDir)
	default:
		return fmt.Errorf("unsupported runnable type: %s", runnable.Type)
	}

	if err != nil {
		return err
	}

	// Update runnable with artifact path
	_, err = w.DB.Exec("UPDATE runnables SET artifact_url = ?, status = 'success' WHERE id = ?",
		artifactPath, runnable.ID)
	if err != nil {
		return err
	}

	// Process deployments for this runnable
	err = w.processDeployments(ctx, runnable, artifactPath)
	if err != nil {
		log.Printf("Failed to process deployments for runnable %s: %v", runnable.Name, err)
		// Don't fail the runnable if deployments fail
	}

	return nil
}

// handleDockerContainer creates and runs a Docker container
func (w *Worker) handleDockerContainer(ctx context.Context, runnable models.Runnable, config models.RunnableConfig, sourceContainerID, tempDir string, job models.Job) (string, error) {
	// Get working directory from config (defaults to /workspace)
	workingDir := config.WorkingDir
	if workingDir == "" {
		workingDir = "/workspace"
	}

	// First, copy the built artifacts from mounted volume to container filesystem
	log.Printf("Copying built artifacts from mounted volume to container filesystem")
	execResp, err := w.Docker.ContainerExecCreate(ctx, sourceContainerID, types.ExecConfig{
		Cmd:          []string{"sh", "-c", "mkdir -p /app && cp -r /workspace/* /app/ && ls -la /app/"},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create exec for copying artifacts: %v", err)
	}

	hijacked, err := w.Docker.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %v", err)
	}

	// Read the copy output
	var output bytes.Buffer
	scanner := bufio.NewScanner(hijacked.Reader)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println("copy output:", line)
		output.WriteString(line + "\n")
	}
	hijacked.Close()

	inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect copy exec: %v", err)
	}
	if inspect.ExitCode != 0 {
		return "", fmt.Errorf("copy failed with exit code %d: %s", inspect.ExitCode, output.String())
	}

	// Now update the working directory to /app and entrypoint accordingly
	actualWorkingDir := "/app"

	// Handle default port exposure if expose_ports is true and no ports specified
	if len(config.Ports) == 0 && job.ExposePorts != nil && *job.ExposePorts {
		// Get environment variables to find PORT setting
		var envs []models.Environment
		err := w.DB.Select(&envs, "SELECT * FROM environments WHERE job_id = ?", job.ID)
		if err == nil {
			for _, env := range envs {
				if env.Key == "PORT" {
					// Use the PORT environment variable as default
					config.Ports = []string{env.Value}
					log.Printf("Using default port from environment: %s", env.Value)
					break
				}
			}
		}

		// If still no port found, use common defaults
		if len(config.Ports) == 0 {
			config.Ports = []string{"3000"} // Default fallback port
			log.Printf("Using fallback default port: 3000")
		}
	}

	// Get entrypoint from config and adjust path if it references /workspace
	var actualEntrypoint []string
	if len(config.Entrypoint) > 0 {
		for _, entry := range config.Entrypoint {
			if entry == "/workspace/server" {
				actualEntrypoint = append(actualEntrypoint, "/app/server")
			} else if strings.HasPrefix(entry, "/workspace/") {
				actualEntrypoint = append(actualEntrypoint, strings.Replace(entry, "/workspace/", "/app/", 1))
			} else {
				actualEntrypoint = append(actualEntrypoint, entry)
			}
		}

		// Make the entrypoint executable
		entrypointPath := actualEntrypoint[0]

		// First check if the entrypoint file exists
		log.Printf("Checking if entrypoint exists: %s", entrypointPath)
		checkResp, err := w.Docker.ContainerExecCreate(ctx, sourceContainerID, types.ExecConfig{
			Cmd:          []string{"sh", "-c", fmt.Sprintf("ls -la %s", entrypointPath)},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create exec for entrypoint check: %v", err)
		}

		checkHijacked, err := w.Docker.ContainerExecAttach(ctx, checkResp.ID, types.ExecStartCheck{})
		if err != nil {
			return "", fmt.Errorf("failed to attach to entrypoint check exec: %v", err)
		}

		var checkOutput bytes.Buffer
		checkScanner := bufio.NewScanner(checkHijacked.Reader)
		for checkScanner.Scan() {
			line := checkScanner.Text()
			log.Printf("entrypoint check output: %s", line)
			checkOutput.WriteString(line + "\n")
		}
		checkHijacked.Close()

		checkInspect, err := w.Docker.ContainerExecInspect(ctx, checkResp.ID)
		if err != nil {
			return "", fmt.Errorf("failed to inspect entrypoint check exec: %v", err)
		}
		if checkInspect.ExitCode != 0 {
			return "", fmt.Errorf("entrypoint file %s does not exist (exit code %d): %s", entrypointPath, checkInspect.ExitCode, checkOutput.String())
		}

		log.Printf("Ensuring entrypoint is executable: %s", entrypointPath)
		execResp, err = w.Docker.ContainerExecCreate(ctx, sourceContainerID, types.ExecConfig{
			Cmd:          []string{"sh", "-c", fmt.Sprintf("chmod +x %s && ls -la %s", entrypointPath, entrypointPath)},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create exec for chmod: %v", err)
		}

		hijacked, err = w.Docker.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
		if err != nil {
			return "", fmt.Errorf("failed to attach to exec: %v", err)
		}
		defer hijacked.Close()

		// Read the chmod output
		output = bytes.Buffer{}
		scanner = bufio.NewScanner(hijacked.Reader)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println("chmod output:", line)
			output.WriteString(line + "\n")
		}

		inspect, err = w.Docker.ContainerExecInspect(ctx, execResp.ID)
		if err != nil {
			return "", fmt.Errorf("failed to inspect chmod exec: %v", err)
		}
		if inspect.ExitCode != 0 {
			return "", fmt.Errorf("chmod failed with exit code %d: %s", inspect.ExitCode, output.String())
		}
	}

	// Determine image name
	imageName := config.ImageName
	if imageName == "" {
		imageName = fmt.Sprintf("rapidflow-job-%d-%s", runnable.JobID, runnable.Name)
	}

	// Create image from current container state
	commitResp, err := w.Docker.ContainerCommit(ctx, sourceContainerID, types.ContainerCommitOptions{
		Reference: imageName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit container: %v", err)
	}

	imageID := commitResp.ID
	log.Printf("Created Docker image: %s with name: %s", imageID, imageName)

	// Create and start new container from committed image
	containerConfig := &container.Config{
		Image: imageID,
		Env:   make([]string, 0),
	}

	// Set working directory to /app (where we copied the artifacts)
	containerConfig.WorkingDir = actualWorkingDir

	// Add environment variables from config
	for key, value := range config.Environment {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set entrypoint from config (adjusted for /app directory)
	if len(actualEntrypoint) > 0 {
		containerConfig.Entrypoint = actualEntrypoint
	}

	// Set exposed ports with Docker-style port mapping support
	var portBindings nat.PortMap
	if len(config.Ports) > 0 {
		containerConfig.ExposedPorts = make(nat.PortSet)
		portBindings = make(nat.PortMap)

		for _, portMapping := range config.Ports {
			hostPort, containerPortStr, hostIP, err := parsePortMapping(portMapping)
			if err != nil {
				return "", fmt.Errorf("failed to parse port mapping '%s': %v", portMapping, err)
			}

			containerPort := nat.Port(fmt.Sprintf("%s/tcp", containerPortStr))
			containerConfig.ExposedPorts[containerPort] = struct{}{}
			portBindings[containerPort] = []nat.PortBinding{{
				HostIP:   hostIP,
				HostPort: hostPort,
			}}

			log.Printf("Port mapping: %s:%s -> %s", hostIP, hostPort, containerPortStr)
		}
	}

	// Determine container name
	containerName := config.ContainerName
	if containerName == "" {
		containerName = fmt.Sprintf("rapidflow-run-%d-%s", runnable.JobID, runnable.Name)
	}

	// Handle existing container with same name by removing it
	err = w.handleExistingContainer(ctx, containerName)
	if err != nil {
		log.Printf("Warning: failed to handle existing container '%s': %v", containerName, err)
		// Don't fail the deployment, just warn
	}

	newContainer, err := w.Docker.ContainerCreate(ctx, containerConfig, &container.HostConfig{
		AutoRemove:   false, // Don't auto-remove so we can track it
		PortBindings: portBindings,
	}, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %v", err)
	}

	// Start container
	err = w.Docker.ContainerStart(ctx, newContainer.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container: %v", err)
	}

	log.Printf("Started Docker container: %s (name: %s)", newContainer.ID, containerName)
	return fmt.Sprintf("container:%s:%s", newContainer.ID, containerName), nil
}

// parsePortMapping parses Docker-style port mappings
// Supports: "3000", "8080:3000", "0.0.0.0:8080:3000"
func parsePortMapping(portStr string) (hostPort, containerPort, hostIP string, err error) {
	parts := strings.Split(portStr, ":")

	switch len(parts) {
	case 1:
		// "3000" - same port on host and container
		containerPort = parts[0]
		hostPort = parts[0]
		hostIP = "0.0.0.0"
	case 2:
		// "8080:3000" - host:container
		hostPort = parts[0]
		containerPort = parts[1]
		hostIP = "0.0.0.0"
	case 3:
		// "0.0.0.0:8080:3000" - hostIP:host:container
		hostIP = parts[0]
		hostPort = parts[1]
		containerPort = parts[2]
	default:
		return "", "", "", fmt.Errorf("invalid port mapping format: %s", portStr)
	}

	return hostPort, containerPort, hostIP, nil
}

// handleExistingContainer removes existing container with the same name if it exists
func (w *Worker) handleExistingContainer(ctx context.Context, containerName string) error {
	// List containers with the same name
	containers, err := w.Docker.ContainerList(ctx, types.ContainerListOptions{
		All: true, // Include stopped containers
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	// Find container with matching name
	for _, container := range containers {
		for _, name := range container.Names {
			// Container names include leading slash, so check for both formats
			if name == "/"+containerName || name == containerName {
				log.Printf("Found existing container '%s' with ID %s, removing it", containerName, container.ID)

				// Remove the container (force will stop it if running)
				err = w.Docker.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
					Force: true, // Force remove even if running
				})
				if err != nil {
					return fmt.Errorf("failed to remove existing container %s: %v", container.ID, err)
				}

				log.Printf("Successfully removed existing container '%s'", containerName)
				return nil
			}
		}
	}

	return nil // No existing container found
}

// handleDockerImage exports Docker image as tar file
func (w *Worker) handleDockerImage(ctx context.Context, runnable models.Runnable, config models.RunnableConfig, sourceContainerID, tempDir string) (string, error) {
	// Determine image name
	imageName := config.ImageName
	if imageName == "" {
		imageName = fmt.Sprintf("rapidflow-job-%d-%s", runnable.JobID, runnable.Name)
	}

	// Create image from current container state
	commitResp, err := w.Docker.ContainerCommit(ctx, sourceContainerID, types.ContainerCommitOptions{
		Reference: imageName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit container: %v", err)
	}

	imageID := commitResp.ID
	imagePath := filepath.Join(tempDir, fmt.Sprintf("%s-image.tar", runnable.Name))

	// Save image to tar file
	err = providers.SaveDockerImage(w.Docker, imageID, imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to save Docker image: %v", err)
	}

	log.Printf("Saved Docker image '%s' to: %s", imageName, imagePath)
	return imagePath, nil
}

// handleArtifacts creates zip archive of workspace
func (w *Worker) handleArtifacts(ctx context.Context, runnable models.Runnable, config models.RunnableConfig, sourceContainerID, tempDir string) (string, error) {
	// Copy workspace from container to local temp directory
	workspaceDir := filepath.Join(tempDir, "workspace")
	err := w.copyFromContainer(ctx, sourceContainerID, "/workspace", workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to copy workspace: %v", err)
	}

	// Create zip archive
	zipPath := filepath.Join(tempDir, fmt.Sprintf("%s-artifacts.zip", runnable.Name))
	err = providers.CreateZipArchive(workspaceDir, zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip archive: %v", err)
	}

	log.Printf("Created artifacts zip: %s", zipPath)
	return zipPath, nil
}

// handleServerless packages for serverless deployment
func (w *Worker) handleServerless(ctx context.Context, runnable models.Runnable, config models.RunnableConfig, sourceContainerID, tempDir string) (string, error) {
	// For serverless, we typically want a zip of the built application
	return w.handleArtifacts(ctx, runnable, config, sourceContainerID, tempDir)
}

// copyFromContainer copies files from container to local filesystem
func (w *Worker) copyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) error {
	reader, _, err := w.Docker.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Extract tar to destination
	return extractTar(reader, dstPath)
}

// extractTar extracts tar archive to destination directory
func extractTar(src io.Reader, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// For simplicity, we'll use a basic approach
	// In production, you'd want proper tar extraction
	tempFile := filepath.Join(dst, "temp.tar")
	outFile, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(outFile, src)
	outFile.Close()

	if err != nil {
		return err
	}

	// Remove temp file
	defer os.Remove(tempFile)

	return nil
}

// processDeployments handles all deployments for a runnable
func (w *Worker) processDeployments(ctx context.Context, runnable models.Runnable, artifactPath string) error {
	// Get deployments for this runnable
	var deployments []models.Deployment
	err := w.DB.Select(&deployments, "SELECT * FROM deployments WHERE runnable_id = ? AND status = 'pending'", runnable.ID)
	if err != nil {
		return err
	}

	log.Printf("Processing %d deployments for runnable %s", len(deployments), runnable.Name)

	for _, deployment := range deployments {
		err = w.processDeployment(ctx, runnable, deployment, artifactPath)
		if err != nil {
			log.Printf("Failed to process deployment %d: %v", deployment.ID, err)
			w.DB.Exec("UPDATE deployments SET status = 'failed', output = ? WHERE id = ?",
				err.Error(), deployment.ID)
			continue
		}

		w.DB.Exec("UPDATE deployments SET status = 'success' WHERE id = ?", deployment.ID)
	}

	return nil
}

// processDeployment handles a single deployment
func (w *Worker) processDeployment(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	log.Printf("Processing deployment: %s", deployment.OutputType)

	// Get provider
	provider, err := w.providerManager.GetProvider(deployment.OutputType)
	if err != nil {
		return err
	}

	// Deploy
	err = provider.Deploy(ctx, runnable, deployment, artifactPath)
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) StartQueue() {
	go func() {
		for {
			// Check for cancelled jobs and clean them up
			var cancelledJobs []models.Job
			err := w.DB.Select(&cancelledJobs, "SELECT * FROM jobs WHERE status = 'running' AND cancelled = 1")
			if err == nil {
				for _, job := range cancelledJobs {
					w.CancelJob(job.ID)
				}
			}

			var jobs []models.Job
			err = w.DB.Select(&jobs, "SELECT id FROM jobs WHERE status = 'pending' ORDER BY created_at ASC LIMIT 1")
			if err != nil {
				log.Printf("Error selecting jobs: %v", err)
				time.Sleep(2 * time.Second) // Wait before retrying
				continue
			}
			if len(jobs) == 0 {
				time.Sleep(1 * time.Second) // Wait before checking again
				continue
			}

			jobID := jobs[0].ID

			// Run job asynchronously (non-blocking)
			go func(id int) {
				err := w.RunJob(id)
				if err != nil {
					log.Printf("Error running job %d: %v", id, err)
					w.DB.Exec("UPDATE jobs SET status = 'failed', finished_at = CURRENT_TIMESTAMP WHERE id = ?", id)
				}
			}(jobID)
		}
	}()
}
