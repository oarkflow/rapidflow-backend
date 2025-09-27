package main

import (
	"database/sql"
	"docker-app/internal/api"
	"docker-app/internal/models"
	"docker-app/internal/worker"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func main() {
	app := &cli.App{
		Name:  "docker-app",
		Usage: "CI/CD platform",
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start the HTTP server",
				Action: func(c *cli.Context) error {
					return startServer()
				},
			},
			{
				Name:  "run-pipeline",
				Usage: "Run a pipeline from YAML file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "Path to pipeline YAML file",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return runPipeline(c.String("file"))
				},
			},
			{
				Name:  "stop-pipeline",
				Usage: "Stop a running pipeline and clean up all resources",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "id",
						Aliases:  []string{"i"},
						Usage:    "Pipeline ID to stop",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return stopPipeline(c.Int("id"))
				},
			},
			{
				Name:  "list-pipelines",
				Usage: "List all pipelines",
				Action: func(c *cli.Context) error {
					return listPipelines()
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startServer() error {
	dir := "./testdata/data"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	// Connect DB
	db, err := sqlx.Connect("sqlite3", "./testdata/data/ci.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Run migrations (simple)
	err = runMigrations(db.DB)
	if err != nil {
		return err
	}

	// Start worker
	w, err := worker.NewWorker(db)
	if err != nil {
		return err
	}
	w.StartQueue()

	// Setup API
	handler := api.NewHandler(db, w)
	app := fiber.New()
	app.Use(cors.New())
	app.Post("/pipelines", handler.CreatePipeline)
	app.Get("/pipelines", handler.GetPipelines)
	app.Get("/pipelines/:id", handler.GetPipeline)
	app.Get("/jobs", handler.GetJobs)
	app.Post("/pipelines/:pipelineID/jobs", handler.CreateJob)
	app.Get("/jobs/:id", handler.GetJob)
	app.Get("/jobs/:id/details", handler.GetJobDetails)
	app.Get("/jobs/:id/logs", handler.GetJobLogs)
	app.Get("/jobs/:id/logs/stream", handler.StreamJobLogs)
	app.Post("/jobs/:id/cancel", handler.CancelJob)
	app.Post("/jobs/:id/retry", handler.RetryJob)
	app.Get("/jobs/:id/steps", handler.GetJobSteps)
	app.Get("/steps/:id", handler.GetStep)
	app.Get("/steps/:id/logs", handler.GetStepLogs)
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("OK") })

	log.Println("Server starting on :3000")
	return app.Listen(":3000")
}

func runMigrations(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS pipelines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    branch TEXT,
    repo_name TEXT,
    repo_url TEXT,
    language TEXT,
    version TEXT,
    folder TEXT,
    expose_ports BOOLEAN DEFAULT 0,
    temporary BOOLEAN DEFAULT 0,
    temp_dir TEXT,
    cancelled BOOLEAN DEFAULT 0,
    container_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id)
);

CREATE TABLE IF NOT EXISTS steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    order_num INTEGER NOT NULL,
    type TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    output TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE TABLE IF NOT EXISTS environments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    step_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    FOREIGN KEY (step_id) REFERENCES steps(id)
);

CREATE TABLE IF NOT EXISTS runnables (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    output TEXT,
    artifact_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE TABLE IF NOT EXISTS deployments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    runnable_id INTEGER NOT NULL,
    output_type TEXT NOT NULL,
    config TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    url TEXT,
    output TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (runnable_id) REFERENCES runnables(id)
);
	`
	_, err := db.Exec(schema)
	return err
}

func runPipeline(filePath string) error {
	// Connect DB
	db, err := sqlx.Connect("sqlite3", "./testdata/data/ci.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Run migrations
	err = runMigrations(db.DB)
	if err != nil {
		return err
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Parse config
	var config models.PipelineConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	// Create pipeline
	pipeline := models.Pipeline{
		Name:   config.Name,
		Config: string(data),
	}
	query := `INSERT INTO pipelines (name, config) VALUES (?, ?)`
	result, err := db.Exec(query, pipeline.Name, pipeline.Config)
	if err != nil {
		return err
	}
	pipelineID, _ := result.LastInsertId()

	// Create job
	job := models.Job{
		PipelineID: int(pipelineID),
		Status:     "pending",
	}
	if config.Branch != "" {
		job.Branch = &config.Branch
	}
	if config.RepoName != "" {
		job.RepoName = &config.RepoName
	}
	if config.RepoURL != "" {
		job.RepoURL = &config.RepoURL
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
	if config.Temporary {
		job.Temporary = &config.Temporary
	}
	query = `INSERT INTO jobs (pipeline_id, status, branch, repo_name, repo_url, language, version, folder, expose_ports, temporary) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err = db.Exec(query, job.PipelineID, job.Status, job.Branch, job.RepoName, job.RepoURL, job.Language, job.Version, job.Folder, job.ExposePorts, job.Temporary)
	if err != nil {
		return err
	}
	jobID, _ := result.LastInsertId()

	// Create steps
	for i, step := range config.Steps {
		result, err := db.Exec(`INSERT INTO steps (job_id, order_num, type, content, status) VALUES (?, ?, ?, ?, ?)`, jobID, i+1, step.Type, step.Content, "pending")
		if err != nil {
			return err
		}
		stepID, _ := result.LastInsertId()
		// Insert files
		for name, content := range step.Files {
			_, err = db.Exec(`INSERT INTO files (step_id, name, content) VALUES (?, ?, ?)`, stepID, name, content)
			if err != nil {
				return err
			}
		}
	}

	// Create env
	for k, v := range config.Env {
		_, err = db.Exec(`INSERT INTO environments (job_id, key, value) VALUES (?, ?, ?)`, jobID, k, v)
		if err != nil {
			return err
		}
	}

	// Create runnables
	for _, runnable := range config.Runnables {
		if !runnable.Enabled {
			continue // Skip disabled runnables
		}

		configJSON, err := json.Marshal(runnable)
		if err != nil {
			return err
		}

		result, err := db.Exec(`INSERT INTO runnables (job_id, name, type, config, status) VALUES (?, ?, ?, ?, ?)`,
			jobID, runnable.Name, runnable.Type, string(configJSON), "pending")
		if err != nil {
			return err
		}

		runnableID, _ := result.LastInsertId()

		// Create deployments for this runnable
		for _, output := range runnable.Outputs {
			outputConfigJSON, err := json.Marshal(output.Config)
			if err != nil {
				return err
			}

			_, err = db.Exec(`INSERT INTO deployments (runnable_id, output_type, config, status) VALUES (?, ?, ?, ?)`,
				runnableID, output.Type, string(outputConfigJSON), "pending")
			if err != nil {
				return err
			}
		}
	}

	log.Printf("Pipeline created and job %d queued", jobID)

	// Start worker and run job synchronously
	w, err := worker.NewWorker(db)
	if err != nil {
		return err
	}
	err = w.RunJob(int(jobID))
	if err != nil {
		log.Printf("Error running job %d: %v", jobID, err)
		db.Exec("UPDATE jobs SET status = 'failed' WHERE id = ?", jobID)
		return err
	}

	return nil
}

func stopPipeline(pipelineID int) error {
	// Connect DB
	db, err := sqlx.Connect("sqlite3", "./testdata/data/ci.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Get all jobs for this pipeline
	var jobs []models.Job
	err = db.Select(&jobs, "SELECT * FROM jobs WHERE pipeline_id = ?", pipelineID)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		log.Printf("No jobs found for pipeline %d", pipelineID)
		return nil
	}

	// Create worker to handle cleanup
	w, err := worker.NewWorker(db)
	if err != nil {
		return err
	}

	log.Printf("Stopping pipeline %d with %d jobs", pipelineID, len(jobs))

	// Stop and clean up each job
	for _, job := range jobs {
		log.Printf("Stopping job %d", job.ID)

		// Cancel any running job
		w.CancelJob(job.ID)

		// Get temp directory from database
		var tempDir string
		if job.TempDir != nil {
			tempDir = *job.TempDir
		}

		var containerID string
		if job.ContainerID != nil {
			containerID = *job.ContainerID
		}

		// Get all runnable containers for this job
		var runnables []models.Runnable
		err = db.Select(&runnables, "SELECT * FROM runnables WHERE job_id = ?", job.ID)
		if err == nil {
			for _, runnable := range runnables {
				var runnableConfig models.RunnableConfig
				if err := json.Unmarshal([]byte(runnable.Config), &runnableConfig); err == nil {
					if runnableConfig.ContainerName != "" {
						log.Printf("Removing runnable container: %s", runnableConfig.ContainerName)
						w.RemoveContainerByName(runnableConfig.ContainerName)
					}
				}
			}
		}

		// Clean up main job container and temp directory
		w.CleanupJobResources(job.ID, containerID, tempDir)

		// Update job status
		db.Exec("UPDATE jobs SET status = 'stopped', finished_at = CURRENT_TIMESTAMP WHERE id = ?", job.ID)
	}

	log.Printf("Pipeline %d stopped and cleaned up successfully", pipelineID)
	return nil
}

func listPipelines() error {
	// Connect DB
	db, err := sqlx.Connect("sqlite3", "./testdata/data/ci.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Get all pipelines with their latest job info
	query := `
		SELECT p.id, p.name, p.created_at,
		       COUNT(j.id) as job_count,
		       MAX(j.created_at) as last_job_time,
		       GROUP_CONCAT(DISTINCT j.status) as job_statuses
		FROM pipelines p
		LEFT JOIN jobs j ON p.id = j.pipeline_id
		GROUP BY p.id, p.name, p.created_at
		ORDER BY p.created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Printf("%-4s %-40s %-8s %-20s %s\n", "ID", "Name", "Jobs", "Last Run", "Status")
	fmt.Println(strings.Repeat("-", 80))

	for rows.Next() {
		var id int
		var name string
		var createdAt string
		var jobCount int
		var lastJobTime *string
		var jobStatuses *string

		err := rows.Scan(&id, &name, &createdAt, &jobCount, &lastJobTime, &jobStatuses)
		if err != nil {
			continue
		}

		lastRun := "Never"
		if lastJobTime != nil {
			lastRun = *lastJobTime
		}

		status := "No jobs"
		if jobStatuses != nil {
			status = *jobStatuses
		}

		fmt.Printf("%-4d %-40s %-8d %-20s %s\n", id, name, jobCount, lastRun, status)
	}

	return nil
}
