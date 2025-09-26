package api

import (
	"docker-app/internal/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type Handler struct {
	DB *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{DB: db}
}

func (h *Handler) CreatePipeline(c *fiber.Ctx) error {
	var pipeline models.Pipeline
	if err := c.BodyParser(&pipeline); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	query := `INSERT INTO pipelines (name, config) VALUES (?, ?)`
	result, err := h.DB.Exec(query, pipeline.Name, pipeline.Config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	id, _ := result.LastInsertId()
	pipeline.ID = int(id)
	return c.Status(201).JSON(pipeline)
}

func (h *Handler) GetPipelines(c *fiber.Ctx) error {
	var pipelines []models.Pipeline
	err := h.DB.Select(&pipelines, "SELECT * FROM pipelines")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(pipelines)
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
	// Parse config
	var config models.PipelineConfig
	// Assume config is JSON for simplicity, but user said YAML, wait.
	// User said configurations, probably YAML.
	// But for API, perhaps accept JSON or YAML.
	// For now, assume JSON.
	err = yaml.Unmarshal([]byte(pipeline.Config), &config)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid config"})
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
	query := `INSERT INTO jobs (pipeline_id, status, branch, repo_name, language, version, folder) VALUES (?, ?, ?, ?, ?, ?, ?)`
	result, err := h.DB.Exec(query, job.PipelineID, job.Status, job.Branch, job.RepoName, job.Language, job.Version, job.Folder)
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
