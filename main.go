package main

import (
	"database/sql"
	"docker-app/internal/api"
	"docker-app/internal/models"
	"docker-app/internal/worker"
	"encoding/json"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
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
	app.Post("/pipelines", handler.CreatePipeline)
	app.Get("/pipelines", handler.GetPipelines)
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
    language TEXT,
    version TEXT,
    folder TEXT,
    expose_ports BOOLEAN DEFAULT 0,
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
	query = `INSERT INTO jobs (pipeline_id, status, branch, repo_name, language, version, folder, expose_ports) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result, err = db.Exec(query, job.PipelineID, job.Status, job.Branch, job.RepoName, job.Language, job.Version, job.Folder, job.ExposePorts)
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
