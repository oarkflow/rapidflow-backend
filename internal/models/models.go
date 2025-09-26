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
}

type StepConfig struct {
	Type    string            `yaml:"type"`    // bash, file, etc.
	Content string            `yaml:"content"` // script or content
	Files   map[string]string `yaml:"files"`   // name: content
}

// JobWithDetails represents a job with all its related data
type JobWithDetails struct {
	Job          Job           `json:"job"`
	Pipeline     Pipeline      `json:"pipeline"`
	Steps        []Step        `json:"steps"`
	Environments []Environment `json:"environments"`
}

// StepWithFiles represents a step with its associated files
type StepWithFiles struct {
	Step  Step   `json:"step"`
	Files []File `json:"files"`
}
