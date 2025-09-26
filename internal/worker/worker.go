package worker

import (
	"bufio"
	"bytes"
	"context"
	"docker-app/internal/models"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/jmoiron/sqlx"
)

type Worker struct {
	DB     *sqlx.DB
	Docker *client.Client
}

func NewWorker(db *sqlx.DB) (*Worker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Worker{DB: db, Docker: cli}, nil
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

func (w *Worker) RunJob(jobID int) error {
	log.Printf("Starting job %d", jobID)
	// Get job
	var job models.Job
	err := w.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", jobID)
	if err != nil {
		return err
	}
	// Update status to running
	_, err = w.DB.Exec("UPDATE jobs SET status = 'running', started_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
	if err != nil {
		return err
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
	ctx := context.Background()
	fallback := false
	// Pull image
	log.Printf("Pulling image %s", baseImage)
	out, err := w.Docker.ImagePull(ctx, baseImage, types.ImagePullOptions{})
	if err != nil {
		log.Printf("Failed to pull image %s: %v, falling back to ubuntu", baseImage, err)
		fallback = true
		baseImage = "ubuntu:latest"
		out, err = w.Docker.ImagePull(ctx, baseImage, types.ImagePullOptions{})
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
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}
	if job.Folder != nil {
		absPath, err := filepath.Abs(*job.Folder)
		if err != nil {
			return err
		}
		hostConfig.Binds = []string{fmt.Sprintf("%s:/workspace", absPath)}
	}
	resp, err := w.Docker.ContainerCreate(ctx, &container.Config{
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
	defer w.Docker.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	// Start container
	err = w.Docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
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
			execResp, err := w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("echo '%s' > /tmp/install.sh && chmod +x /tmp/install.sh", string(scriptContent))},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
			if err != nil {
				return err
			}
			if inspect.ExitCode != 0 {
				return fmt.Errorf("failed to create install script")
			}
			// Run script
			execResp, err = w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
				Cmd:          []string{"/tmp/install.sh"},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			hijacked, err := w.Docker.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
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
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			inspect, err = w.Docker.ContainerExecInspect(ctx, execResp.ID)
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
		execResp, err := w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
			Cmd:          []string{"sh", "-c", fmt.Sprintf("git clone %s /workspace", *job.RepoName)},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			return err
		}
		err = w.Docker.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
		if err != nil {
			return err
		}
		inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
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
			execResp, err := w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("cd /workspace && git checkout %s", *job.Branch)},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
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
			execResp, err := w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", fmt.Sprintf("echo '%s' > %s", content, f.Name)},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			err = w.Docker.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{})
			if err != nil {
				return err
			}
			// Wait for exec
			inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
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
			execResp, err := w.Docker.ContainerExecCreate(ctx, containerID, types.ExecConfig{
				Cmd:          []string{"sh", "-c", step.Content},
				AttachStdout: true,
				AttachStderr: true,
			})
			if err != nil {
				return err
			}
			hijacked, err := w.Docker.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
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
			}
			if err := scanner.Err(); err != nil {
				return err
			}
			// Wait for exec
			inspect, err := w.Docker.ContainerExecInspect(ctx, execResp.ID)
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
		}
	}
	// Update job status
	_, err = w.DB.Exec("UPDATE jobs SET status = 'success', finished_at = CURRENT_TIMESTAMP WHERE id = ?", jobID)
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) StartQueue() {
	go func() {
		for {
			var jobs []models.Job
			err := w.DB.Select(&jobs, "SELECT id FROM jobs WHERE status = 'pending' LIMIT 1")
			if err != nil {
				log.Printf("Error selecting jobs: %v", err)
				continue
			}
			if len(jobs) == 0 {
				continue
			}
			jobID := jobs[0].ID
			go func(id int) {
				err := w.RunJob(id)
				if err != nil {
					log.Printf("Error running job %d: %v", id, err)
					w.DB.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", id)
				}
			}(jobID)
		}
	}()
}
