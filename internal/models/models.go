package models

import (
	"time"
)

type Pipeline struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Config    string    `db:"config" json:"config"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Job struct {
	ID          int        `db:"id" json:"id"`
	PipelineID  int        `db:"pipeline_id" json:"pipeline_id"`
	Status      string     `db:"status" json:"status"`
	Branch      *string    `db:"branch" json:"branch"`
	RepoName    *string    `db:"repo_name" json:"repo_name"`
	RepoURL     *string    `db:"repo_url" json:"repo_url"`
	Language    *string    `db:"language" json:"language"`
	Version     *string    `db:"version" json:"version"`
	Folder      *string    `db:"folder" json:"folder"`
	ExposePorts *bool      `db:"expose_ports" json:"expose_ports"`
	Temporary   *bool      `db:"temporary" json:"temporary"`
	TempDir     *string    `db:"temp_dir" json:"temp_dir"`
	Cancelled   bool       `db:"cancelled" json:"cancelled"`
	ContainerID *string    `db:"container_id" json:"container_id"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at"`
	FinishedAt  *time.Time `db:"finished_at" json:"finished_at"`
}

type Step struct {
	ID        int       `db:"id" json:"id"`
	JobID     int       `db:"job_id" json:"job_id"`
	OrderNum  int       `db:"order_num" json:"order_num"`
	Type      string    `db:"type" json:"type"`
	Content   string    `db:"content" json:"content"`
	Status    string    `db:"status" json:"status"`
	Output    *string   `db:"output" json:"output"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Environment struct {
	ID    int    `db:"id" json:"id"`
	JobID int    `db:"job_id" json:"job_id"`
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
}

type File struct {
	ID      int    `db:"id" json:"id"`
	StepID  int    `db:"step_id" json:"step_id"`
	Name    string `db:"name" json:"name"`
	Content string `db:"content" json:"content"`
}

type PipelineConfig struct {
	Name        string            `yaml:"name"`
	Language    string            `yaml:"language,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Branch      string            `yaml:"branch,omitempty"`
	RepoName    string            `yaml:"repo_name,omitempty"`
	RepoURL     string            `yaml:"repo_url,omitempty"`
	Folder      string            `yaml:"folder,omitempty"`
	ExposePorts bool              `yaml:"expose_ports,omitempty"`
	Temporary   bool              `yaml:"temporary,omitempty"`
	Env         map[string]string `yaml:"env"`
	Steps       []StepConfig      `yaml:"steps"`
	Runnables   []RunnableConfig  `yaml:"runnables,omitempty"`
}

type StepConfig struct {
	Type    string            `yaml:"type"`
	Content string            `yaml:"content"`
	Files   map[string]string `yaml:"files"`
}

type RunnableConfig struct {
	Type          string                 `yaml:"type"`
	Name          string                 `yaml:"name"`
	Enabled       bool                   `yaml:"enabled"`
	Config        map[string]interface{} `yaml:"config"`
	Outputs       []OutputConfig         `yaml:"outputs"`
	Dockerfile    string                 `yaml:"dockerfile"`
	Entrypoint    []string               `yaml:"entrypoint"`
	Ports         []string               `yaml:"ports"`
	Environment   map[string]string      `yaml:"environment"`
	ContainerName string                 `yaml:"container_name"`
	ImageName     string                 `yaml:"image_name"`
	WorkingDir    string                 `yaml:"working_dir"`
}

type OutputConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

type Runnable struct {
	ID          int       `db:"id" json:"id"`
	JobID       int       `db:"job_id" json:"job_id"`
	Name        string    `db:"name" json:"name"`
	Type        string    `db:"type" json:"type"`
	Config      string    `db:"config" json:"config"`
	Status      string    `db:"status" json:"status"`
	Output      *string   `db:"output" json:"output"`
	ArtifactURL *string   `db:"artifact_url" json:"artifact_url"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Deployment struct {
	ID         int       `db:"id" json:"id"`
	RunnableID int       `db:"runnable_id" json:"runnable_id"`
	OutputType string    `db:"output_type" json:"output_type"`
	Config     string    `db:"config" json:"config"`
	Status     string    `db:"status" json:"status"`
	URL        *string   `db:"url" json:"url"`
	Output     *string   `db:"output" json:"output"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type JobWithDetails struct {
	Job          Job           `json:"job"`
	Pipeline     Pipeline      `json:"pipeline"`
	Steps        []Step        `json:"steps"`
	Environments []Environment `json:"environments"`
	Runnables    []Runnable    `json:"runnables"`
	Deployments  []Deployment  `json:"deployments"`
}
