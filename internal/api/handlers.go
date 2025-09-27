package api

import (
	"docker-app/internal/models"
	"docker-app/internal/worker"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/oarkflow/bcl"
	"gopkg.in/yaml.v3"
)

// ConfigFormat represents the format of pipeline configuration
type ConfigFormat string

const (
	FormatYAML ConfigFormat = "yaml"
	FormatJSON ConfigFormat = "json"
	FormatBCL  ConfigFormat = "bcl"
)

// detectConfigFormat detects the format of the configuration string
func detectConfigFormat(config string) ConfigFormat {
	config = strings.TrimSpace(config)

	// Check for JSON (starts with { or [)
	if strings.HasPrefix(config, "{") || strings.HasPrefix(config, "[") {
		return FormatJSON
	}

	// Check for YAML (contains : or - at beginning of lines)
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, ":") || strings.HasPrefix(line, "-") {
			return FormatYAML
		}
	}

	// Default to YAML for backward compatibility
	return FormatYAML
}

// unmarshalConfig unmarshals the configuration string based on detected format
func unmarshalConfig(configStr string, config *models.PipelineConfig) error {
	format := detectConfigFormat(configStr)

	switch format {
	case FormatJSON:
		return json.Unmarshal([]byte(configStr), config)
	case FormatBCL:
		// For BCL, we need to use UnmarshalJSON after parsing
		// First parse the BCL to get AST nodes, then convert to JSON and unmarshal
		nodes, err := bcl.Unmarshal([]byte(configStr), config)
		if err != nil {
			return err
		}
		// If nodes are returned but we want to unmarshal into config, we might need a different approach
		// For now, let's try to use the config directly if it was modified
		if len(nodes) > 0 {
			// Convert to JSON and then unmarshal
			jsonData, err := bcl.MarshalJSON(config)
			if err != nil {
				return err
			}
			return json.Unmarshal(jsonData, config)
		}
		return nil
	case FormatYAML:
		fallthrough
	default:
		return yaml.Unmarshal([]byte(configStr), config)
	}
}

type Handler struct {
	DB     *sqlx.DB
	Worker *worker.Worker
}

func NewHandler(db *sqlx.DB, w *worker.Worker) *Handler {
	return &Handler{DB: db, Worker: w}
}

func (h *Handler) CreatePipeline(c *fiber.Ctx) error {
	var config models.PipelineConfig
	if err := c.BodyParser(&config); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	// Convert config to YAML
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to marshal config"})
	}
	query := `INSERT INTO pipelines (name, config) VALUES (?, ?)`
	result, err := h.DB.Exec(query, config.Name, string(configYAML))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	id, _ := result.LastInsertId()
	pipeline := models.Pipeline{
		ID:     int(id),
		Name:   config.Name,
		Config: string(configYAML),
	}
	return c.Status(201).JSON(pipeline)
}

func (h *Handler) GetPipelines(c *fiber.Ctx) error {
	var pipelines []models.Pipeline
	err := h.DB.Select(&pipelines, "SELECT * FROM pipelines")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Unmarshal config for each pipeline
	for i, pipeline := range pipelines {
		var config models.PipelineConfig
		if err := unmarshalConfig(pipeline.Config, &config); err != nil {
			// If unmarshaling fails, keep the raw config but log the error
			log.Printf("Failed to unmarshal config for pipeline %d: %v", pipeline.ID, err)
			continue
		}
		configBytes, err := json.Marshal(config)
		if err != nil {
			log.Printf("Failed to marshal config to JSON for pipeline %d: %v", pipeline.ID, err)
			continue
		}
		pipeline.Config = string(configBytes)
		pipelines[i] = pipeline
	}

	return c.JSON(pipelines)
}

func (h *Handler) GetPipeline(c *fiber.Ctx) error {
	id := c.Params("id")
	var pipeline models.Pipeline
	err := h.DB.Get(&pipeline, "SELECT * FROM pipelines WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Pipeline not found"})
	}

	// Try to unmarshal the config to validate format
	var config models.PipelineConfig
	if err := unmarshalConfig(pipeline.Config, &config); err != nil {
		log.Printf("Failed to unmarshal config for pipeline %d: %v", pipeline.ID, err)
		// Return pipeline with raw config and error info
		return c.JSON(fiber.Map{
			"pipeline":            pipeline,
			"config_format_error": err.Error(),
		})
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Printf("Failed to marshal config to JSON for pipeline %d: %v", pipeline.ID, err)
		return c.JSON(fiber.Map{
			"pipeline":            pipeline,
			"config_format_error": err.Error(),
		})
	}
	pipeline.Config = string(configBytes)

	// Return pipeline with parsed config info
	return c.JSON(pipeline)
}

func (h *Handler) GetPipelineJobs(c *fiber.Ctx) error {
	pipelineID := c.Params("id")
	var jobs []models.Job
	err := h.DB.Select(&jobs, "SELECT * FROM jobs WHERE pipeline_id = ? ORDER BY created_at DESC", pipelineID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(jobs)
}

func (h *Handler) GetJobs(c *fiber.Ctx) error {
	var jobs []models.Job
	err := h.DB.Select(&jobs, "SELECT * FROM jobs")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(jobs)
}

func (h *Handler) CreateJob(c *fiber.Ctx) error {
	pipelineIDStr := c.Params("pipelineID")
	pipelineID, err := strconv.Atoi(pipelineIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid pipeline id"})
	}
	var pipeline models.Pipeline
	err = h.DB.Get(&pipeline, "SELECT * FROM pipelines WHERE id = ?", pipelineID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "pipeline not found"})
	}
	// Parse config using format detection
	var config models.PipelineConfig
	err = unmarshalConfig(pipeline.Config, &config)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid config: " + err.Error()})
	}
	// Create job
	job := models.Job{
		PipelineID: pipelineID,
		Status:     "pending",
	}
	if config.Branch != "" {
		job.Branch = &config.Branch
	}
	if config.RepoName != "" {
		job.RepoName = &config.RepoName
	}
	if config.Language != "" {
		job.Language = &config.Language
	}
	if config.Version != "" {
		job.Version = &config.Version
	}
	if config.Folder != "" {
		job.Folder = &config.Folder
	}
	if config.ExposePorts {
		job.ExposePorts = &config.ExposePorts
	}
	query := `INSERT INTO jobs (pipeline_id, status, branch, repo_name, language, version, folder, expose_ports) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := h.DB.Exec(query, job.PipelineID, job.Status, job.Branch, job.RepoName, job.Language, job.Version, job.Folder, job.ExposePorts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	id, _ := result.LastInsertId()
	job.ID = int(id)
	// Create steps
	for i, step := range config.Steps {
		result, err := h.DB.Exec(`INSERT INTO steps (job_id, order_num, type, content, status) VALUES (?, ?, ?, ?, ?)`, job.ID, i+1, step.Type, step.Content, "pending")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		stepID, _ := result.LastInsertId()
		// Insert files
		for name, content := range step.Files {
			_, err = h.DB.Exec(`INSERT INTO files (step_id, name, content) VALUES (?, ?, ?)`, stepID, name, content)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
	}
	// For env
	for k, v := range config.Env {
		_, err = h.DB.Exec(`INSERT INTO environments (job_id, key, value) VALUES (?, ?, ?)`, job.ID, k, v)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Create runnables
	for _, runnable := range config.Runnables {
		if !runnable.Enabled {
			continue // Skip disabled runnables
		}

		configJSON, err := json.Marshal(runnable)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		result, err := h.DB.Exec(`INSERT INTO runnables (job_id, name, type, config, status) VALUES (?, ?, ?, ?, ?)`,
			job.ID, runnable.Name, runnable.Type, string(configJSON), "pending")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		runnableID, _ := result.LastInsertId()

		// Create deployments for this runnable
		for _, output := range runnable.Outputs {
			outputConfigJSON, err := json.Marshal(output.Config)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			_, err = h.DB.Exec(`INSERT INTO deployments (runnable_id, output_type, config, status) VALUES (?, ?, ?, ?)`,
				runnableID, output.Type, string(outputConfigJSON), "pending")
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
	}

	return c.Status(201).JSON(job)
}

func (h *Handler) GetJob(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var job models.Job
	err = h.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}
	return c.JSON(job)
}

func (h *Handler) GetJobSteps(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var steps []models.Step
	err = h.DB.Select(&steps, "SELECT * FROM steps WHERE job_id = ? ORDER BY order_num", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(steps)
}

func (h *Handler) GetStep(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	var step models.Step
	err = h.DB.Get(&step, "SELECT * FROM steps WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "step not found"})
	}
	return c.JSON(step)
}

// GetJobDetails returns detailed job information with all related data
func (h *Handler) GetJobDetails(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	// Get job
	var job models.Job
	err = h.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	// Get pipeline
	var pipeline models.Pipeline
	err = h.DB.Get(&pipeline, "SELECT * FROM pipelines WHERE id = ?", job.PipelineID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "pipeline not found"})
	}

	// Get steps
	var steps []models.Step
	err = h.DB.Select(&steps, "SELECT * FROM steps WHERE job_id = ? ORDER BY order_num", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Get environments
	var environments []models.Environment
	err = h.DB.Select(&environments, "SELECT * FROM environments WHERE job_id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Get runnables
	var runnables []models.Runnable
	err = h.DB.Select(&runnables, "SELECT * FROM runnables WHERE job_id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Get deployments for all runnables
	var deployments []models.Deployment
	if len(runnables) > 0 {
		runnableIDs := make([]int, len(runnables))
		for i, r := range runnables {
			runnableIDs[i] = r.ID
		}

		// Build IN clause for SQL
		inClause := strings.Repeat("?,", len(runnableIDs)-1) + "?"
		query := fmt.Sprintf("SELECT * FROM deployments WHERE runnable_id IN (%s)", inClause)

		args := make([]interface{}, len(runnableIDs))
		for i, id := range runnableIDs {
			args[i] = id
		}

		err = h.DB.Select(&deployments, query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	details := models.JobWithDetails{
		Job:          job,
		Pipeline:     pipeline,
		Steps:        steps,
		Environments: environments,
		Runnables:    runnables,
		Deployments:  deployments,
	}

	return c.JSON(details)
}

// CancelJob cancels a running job
func (h *Handler) CancelJob(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	// Get job
	var job models.Job
	err = h.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	// Check if job is in a cancellable state
	if job.Status != "running" && job.Status != "pending" {
		return c.Status(400).JSON(fiber.Map{"error": "job cannot be cancelled", "status": job.Status})
	}

	// Update job status to cancelled in database
	_, err = h.DB.Exec("UPDATE jobs SET cancelled = 1 WHERE id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Try to cancel the running job if it's currently running
	if job.Status == "running" && h.Worker != nil {
		err = h.Worker.CancelJob(id)
		if err != nil {
			// Job might not be running anymore, which is fine
			log.Printf("Could not cancel running job %d: %v", id, err)
		}
	}

	// Update the final status
	_, err = h.DB.Exec("UPDATE jobs SET status = 'cancelled', finished_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update any pending/running steps to cancelled
	_, err = h.DB.Exec("UPDATE steps SET status = 'cancelled' WHERE job_id = ? AND status IN ('pending', 'running')", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "job cancelled successfully"})
}

// RetryJob creates a new job based on an existing job
func (h *Handler) RetryJob(c *fiber.Ctx) error {
	idStr := c.Params("id")
	originalJobID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	// Get original job
	var originalJob models.Job
	err = h.DB.Get(&originalJob, "SELECT * FROM jobs WHERE id = ?", originalJobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	// Only allow retrying completed jobs
	if originalJob.Status == "running" || originalJob.Status == "pending" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot retry running or pending job"})
	}

	// Create new job with same parameters
	query := `INSERT INTO jobs (pipeline_id, status, branch, repo_name, language, version, folder, expose_ports) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := h.DB.Exec(query, originalJob.PipelineID, "pending", originalJob.Branch, originalJob.RepoName, originalJob.Language, originalJob.Version, originalJob.Folder, originalJob.ExposePorts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	newJobID, _ := result.LastInsertId()

	// Copy steps from original job
	var originalSteps []models.Step
	err = h.DB.Select(&originalSteps, "SELECT * FROM steps WHERE job_id = ? ORDER BY order_num", originalJobID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	for _, step := range originalSteps {
		result, err := h.DB.Exec(`INSERT INTO steps (job_id, order_num, type, content, status) VALUES (?, ?, ?, ?, ?)`, newJobID, step.OrderNum, step.Type, step.Content, "pending")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		newStepID, _ := result.LastInsertId()

		// Copy files for this step
		var files []models.File
		err = h.DB.Select(&files, "SELECT * FROM files WHERE step_id = ?", step.ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		for _, file := range files {
			_, err = h.DB.Exec(`INSERT INTO files (step_id, name, content) VALUES (?, ?, ?)`, newStepID, file.Name, file.Content)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
	}

	// Copy environments
	var environments []models.Environment
	err = h.DB.Select(&environments, "SELECT * FROM environments WHERE job_id = ?", originalJobID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	for _, env := range environments {
		_, err = h.DB.Exec(`INSERT INTO environments (job_id, key, value) VALUES (?, ?, ?)`, newJobID, env.Key, env.Value)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Get the new job to return
	var newJob models.Job
	err = h.DB.Get(&newJob, "SELECT * FROM jobs WHERE id = ?", newJobID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(newJob)
}

// GetJobLogs returns the logs for a specific job (all steps combined)
func (h *Handler) GetJobLogs(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	// Get job to verify it exists
	var job models.Job
	err = h.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	// Get all steps with their outputs
	var steps []models.Step
	err = h.DB.Select(&steps, "SELECT * FROM steps WHERE job_id = ? ORDER BY order_num", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Combine all logs with step information
	var logs []map[string]interface{}
	for _, step := range steps {
		logEntry := map[string]interface{}{
			"step_id":    step.ID,
			"order_num":  step.OrderNum,
			"type":       step.Type,
			"content":    step.Content,
			"status":     step.Status,
			"output":     step.Output,
			"created_at": step.CreatedAt,
		}
		logs = append(logs, logEntry)
	}

	return c.JSON(fiber.Map{
		"job_id": id,
		"status": job.Status,
		"logs":   logs,
	})
}

// GetStepLogs returns the logs for a specific step
func (h *Handler) GetStepLogs(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid step id"})
	}

	// Get step
	var step models.Step
	err = h.DB.Get(&step, "SELECT * FROM steps WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "step not found"})
	}

	return c.JSON(fiber.Map{
		"step_id":    step.ID,
		"job_id":     step.JobID,
		"order_num":  step.OrderNum,
		"type":       step.Type,
		"content":    step.Content,
		"status":     step.Status,
		"output":     step.Output,
		"created_at": step.CreatedAt,
	})
}

// StreamJobLogs streams the logs for a job (useful for real-time monitoring)
func (h *Handler) StreamJobLogs(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	// Get job to verify it exists
	var job models.Job
	err = h.DB.Get(&job, "SELECT * FROM jobs WHERE id = ?", id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	// Set headers for streaming
	c.Set("Content-Type", "text/plain")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	// Get query parameters
	follow := c.Query("follow", "false") == "true"
	sinceStr := c.Query("since")
	var since *int
	if sinceStr != "" {
		if sinceInt, err := strconv.Atoi(sinceStr); err == nil {
			since = &sinceInt
		}
	}

	// Build query based on parameters
	query := "SELECT * FROM steps WHERE job_id = ?"
	args := []interface{}{id}

	if since != nil {
		query += " AND id > ?"
		args = append(args, *since)
	}

	query += " ORDER BY order_num, id"

	// Get initial logs
	var steps []models.Step
	err = h.DB.Select(&steps, query, args...)
	if err != nil {
		return c.Status(500).SendString("Error retrieving logs: " + err.Error())
	}

	// Send initial logs
	for _, step := range steps {
		if step.Output != nil && *step.Output != "" {
			c.WriteString(fmt.Sprintf("=== Step %d (%s) ===\n", step.OrderNum, step.Type))
			c.WriteString(*step.Output)
			c.WriteString("\n")
		}
	}

	// If follow is requested and job is still running, keep polling
	if follow && (job.Status == "running" || job.Status == "pending") {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastStepID := 0
		if len(steps) > 0 {
			lastStepID = steps[len(steps)-1].ID
		}

		for {
			select {
			case <-ticker.C:
				// Check if job is still running
				err = h.DB.Get(&job, "SELECT status FROM jobs WHERE id = ?", id)
				if err != nil || (job.Status != "running" && job.Status != "pending") {
					c.WriteString("\n=== Build completed ===\n")
					return nil
				}

				// Get new logs since last check
				var newSteps []models.Step
				err = h.DB.Select(&newSteps, "SELECT * FROM steps WHERE job_id = ? AND id > ? ORDER BY order_num, id", id, lastStepID)
				if err != nil {
					continue
				}

				// Send new logs
				for _, step := range newSteps {
					if step.Output != nil && *step.Output != "" {
						c.WriteString(fmt.Sprintf("=== Step %d (%s) ===\n", step.OrderNum, step.Type))
						c.WriteString(*step.Output)
						c.WriteString("\n")
						lastStepID = step.ID
					}
				}

			case <-c.Context().Done():
				return nil
			}
		}
	}

	return nil
}
