package models

import (
	"time"
)

type Pipeline struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Config    string    `db:"config"`
	CreatedAt time.Time `db:"created_at"`
}

type Job struct {
	ID          int        `db:"id"`
	PipelineID  int        `db:"pipeline_id"`
	Status      string     `db:"status"`
	Branch      *string    `db:"branch"`
	RepoName    *string    `db:"repo_name"`
	Language    *string    `db:"language"`
	Version     *string    `db:"version"`
	Folder      *string    `db:"folder"`
	ExposePorts *bool      `db:"expose_ports"`
	Cancelled   bool       `db:"cancelled"`
	ContainerID *string    `db:"container_id"`
	CreatedAt   time.Time  `db:"created_at"`
	StartedAt   *time.Time `db:"started_at"`
	FinishedAt  *time.Time `db:"finished_at"`
}

type Step struct {
	ID        int       `db:"id"`
	JobID     int       `db:"job_id"`
	OrderNum  int       `db:"order_num"`
	Type      string    `db:"type"`
	Content   string    `db:"content"`
	Status    string    `db:"status"`
	Output    *string   `db:"output"`
	CreatedAt time.Time `db:"created_at"`
}

type Environment struct {
	ID    int    `db:"id"`
	JobID int    `db:"job_id"`
	Key   string `db:"key"`
	Value string `db:"value"`
}

type File struct {
	ID      int    `db:"id"`
	StepID  int    `db:"step_id"`
	Name    string `db:"name"`
	Content string `db:"content"`
}

type PipelineConfig struct {
	Name        string            `yaml:"name"`
	Language    string            `yaml:"language"`
	Version     string            `yaml:"version,omitempty"`
	Branch      string            `yaml:"branch,omitempty"`
	RepoName    string            `yaml:"repo_name,omitempty"`
	Folder      string            `yaml:"folder,omitempty"`
	ExposePorts bool              `yaml:"expose_ports,omitempty"`
	Env         map[string]string `yaml:"env"`
	Steps       []StepConfig      `yaml:"steps"`
	Runnables   []RunnableConfig  `yaml:"runnables,omitempty"`
}

type StepConfig struct {
	Type    string            `yaml:"type"`    // bash, file, etc.
	Content string            `yaml:"content"` // script or content
	Files   map[string]string `yaml:"files"`   // name: content
}

// RunnableConfig defines how to package/deploy the built application
type RunnableConfig struct {
	Type          string                 `yaml:"type"`           // docker_container, docker_image, artifacts, serverless
	Name          string                 `yaml:"name"`           // runnable name
	Enabled       bool                   `yaml:"enabled"`        // whether this runnable is active
	Config        map[string]interface{} `yaml:"config"`         // type-specific configuration
	Outputs       []OutputConfig         `yaml:"outputs"`        // where to send the runnable
	Dockerfile    string                 `yaml:"dockerfile"`     // custom dockerfile content
	Entrypoint    []string               `yaml:"entrypoint"`     // custom entrypoint
	Ports         []string               `yaml:"ports"`          // Docker-style port mappings: ["3000"], ["3001:3000"], etc.
	Environment   map[string]string      `yaml:"environment"`    // runtime environment variables
	ContainerName string                 `yaml:"container_name"` // custom container name
	ImageName     string                 `yaml:"image_name"`     // custom image name/tag
	WorkingDir    string                 `yaml:"working_dir"`    // working directory in container
}

// OutputConfig defines where to send/deploy the runnable
type OutputConfig struct {
	Type   string                 `yaml:"type"`   // s3, email, registry, local, webhook
	Config map[string]interface{} `yaml:"config"` // output-specific configuration
}

// Database models for runnables
type Runnable struct {
	ID          int       `db:"id"`
	JobID       int       `db:"job_id"`
	Name        string    `db:"name"`
	Type        string    `db:"type"`
	Config      string    `db:"config"`       // JSON config
	Status      string    `db:"status"`       // pending, running, success, failed
	Output      *string   `db:"output"`       // execution output
	ArtifactURL *string   `db:"artifact_url"` // URL to generated artifact
	CreatedAt   time.Time `db:"created_at"`
}

type Deployment struct {
	ID         int       `db:"id"`
	RunnableID int       `db:"runnable_id"`
	OutputType string    `db:"output_type"` // s3, email, etc.
	Config     string    `db:"config"`      // JSON config
	Status     string    `db:"status"`      // pending, success, failed
	URL        *string   `db:"url"`         // deployment URL or reference
	Output     *string   `db:"output"`      // deployment output/logs
	CreatedAt  time.Time `db:"created_at"`
}

// JobWithDetails represents a job with all its related data
type JobWithDetails struct {
	Job          Job           `json:"job"`
	Pipeline     Pipeline      `json:"pipeline"`
	Steps        []Step        `json:"steps"`
	Environments []Environment `json:"environments"`
	Runnables    []Runnable    `json:"runnables,omitempty"`
	Deployments  []Deployment  `json:"deployments,omitempty"`
}

// StepWithFiles represents a step with its associated files
type StepWithFiles struct {
	Step  Step   `json:"step"`
	Files []File `json:"files"`
}
