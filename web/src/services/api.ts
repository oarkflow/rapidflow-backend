const API_BASE_URL = 'http://localhost:3000';

export interface Pipeline {
    id: number;
    name: string;
    config: string;
    created_at: string;
}

export interface Job {
    id: number;
    pipeline_id: number;
    status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'stopped';
    branch?: string;
    repo_name?: string;
    repo_url?: string;
    language?: string;
    version?: string;
    folder?: string;
    expose_ports?: boolean;
    temporary?: boolean;
    temp_dir?: string;
    cancelled?: boolean;
    container_id?: string;
    created_at: string;
    started_at?: string;
    finished_at?: string;
}

export interface Step {
    id: number;
    job_id: number;
    order_num: number;
    type: string;
    content: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    output?: string;
    created_at: string;
}

export interface Environment {
    id: number;
    job_id: number;
    key: string;
    value: string;
}

export interface Runnable {
    id: number;
    job_id: number;
    name: string;
    type: string;
    config: string;
    status: string;
    output?: string;
    artifact_url?: string;
    created_at: string;
}

export interface Deployment {
    id: number;
    runnable_id: number;
    output_type: string;
    config: string;
    status: string;
    url?: string;
    output?: string;
    created_at: string;
}

export interface JobWithDetails {
    job: Job;
    pipeline: Pipeline;
    steps: Step[];
    environments: Environment[];
    runnables: Runnable[];
    deployments: Deployment[];
}

export interface PipelineConfig {
    name: string;
    language?: string;
    version?: string;
    branch?: string;
    folder?: string;
    repo_url?: string;
    expose_ports?: boolean;
    temporary?: boolean;
    env?: Record<string, string>;
    steps: {
        type: string;
        content: string;
        files?: Record<string, string>;
    }[];
    runnables?: {
        name: string;
        type: string;
        enabled: boolean;
        container_name?: string;
        image_name?: string;
        entrypoint?: string[];
        working_dir?: string;
        ports?: string[];
        environment?: Record<string, string>;
        config?: Record<string, any>;
        outputs?: {
            type: string;
            config: Record<string, any>;
        }[];
    }[];
}

// Mock API service for development
class MockApiService {
    private pipelines: Pipeline[] = [
        {
            id: 1,
            name: "Frontend Build Pipeline",
            config: JSON.stringify({
                name: "Frontend Build Pipeline",
                language: "nodejs",
                version: "18",
                steps: [
                    { type: "bash", content: "npm install" },
                    { type: "bash", content: "npm run build" }
                ]
            }),
            created_at: new Date().toISOString()
        },
        {
            id: 2,
            name: "Backend API Pipeline",
            config: JSON.stringify({
                name: "Backend API Pipeline",
                language: "golang",
                version: "1.21",
                steps: [
                    { type: "bash", content: "go mod download" },
                    { type: "bash", content: "go build -o server ." }
                ]
            }),
            created_at: new Date().toISOString()
        }
    ];

    private jobs: Job[] = [
        {
            id: 1,
            pipeline_id: 1,
            status: 'completed',
            branch: 'main',
            language: 'nodejs',
            version: '18',
            created_at: new Date().toISOString()
        },
        {
            id: 2,
            pipeline_id: 2,
            status: 'running',
            branch: 'develop',
            language: 'golang',
            version: '1.21',
            created_at: new Date().toISOString()
        }
    ];

    async getPipelines(): Promise<Pipeline[]> {
        await this.delay(500);
        return this.pipelines;
    }

    async getPipeline(id: number): Promise<Pipeline | null> {
        await this.delay(300);
        return this.pipelines.find(p => p.id === id) || null;
    }

    async createPipeline(config: PipelineConfig): Promise<Pipeline> {
        await this.delay(800);
        const pipeline: Pipeline = {
            id: this.pipelines.length + 1,
            name: config.name,
            config: JSON.stringify(config),
            created_at: new Date().toISOString()
        };
        this.pipelines.push(pipeline);
        return pipeline;
    }

    async getJobs(): Promise<Job[]> {
        await this.delay(400);
        return this.jobs;
    }

    async getJobsByPipeline(pipelineId: number): Promise<Job[]> {
        await this.delay(300);
        return this.jobs.filter(job => job.pipeline_id === pipelineId);
    }

    async getJobDetails(jobId: number): Promise<JobWithDetails | null> {
        await this.delay(400);
        const job = this.jobs.find(j => j.id === jobId);
        if (!job) return null;

        const pipeline = this.pipelines.find(p => p.id === job.pipeline_id);
        if (!pipeline) return null;

        return {
            job,
            pipeline,
            steps: [
                { id: 1, job_id: jobId, order_num: 1, type: 'bash', content: 'npm install', status: 'completed', output: 'Dependencies installed successfully', created_at: new Date().toISOString() },
                { id: 2, job_id: jobId, order_num: 2, type: 'bash', content: 'npm run build', status: job.status === 'running' ? 'running' : 'completed', output: 'Build completed', created_at: new Date().toISOString() }
            ],
            environments: [
                { id: 1, job_id: jobId, key: 'NODE_ENV', value: 'production' },
                { id: 2, job_id: jobId, key: 'PORT', value: '3000' }
            ],
            runnables: [],
            deployments: []
        };
    }

    async getJobLogs(jobId: number): Promise<string[]> {
        await this.delay(200);
        const logs = [
            `[${new Date().toISOString()}] Starting job #${jobId}`,
            `[${new Date().toISOString()}] Initializing environment...`,
            `[${new Date().toISOString()}] Running step 1: npm install`,
            `[${new Date().toISOString()}] Dependencies installed successfully`,
            `[${new Date().toISOString()}] Running step 2: npm run build`,
        ];

        if (Math.random() > 0.5) {
            logs.push(`[${new Date().toISOString()}] Build completed successfully`);
            logs.push(`[${new Date().toISOString()}] Job #${jobId} completed`);
        }

        return logs;
    }

    async createJob(pipelineId: number): Promise<Job> {
        await this.delay(600);
        const job: Job = {
            id: this.jobs.length + 1,
            pipeline_id: pipelineId,
            status: 'pending',
            created_at: new Date().toISOString()
        };
        this.jobs.push(job);
        return job;
    }

    private delay(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// For production, replace with actual API calls
class ApiService {
    async getPipelines(): Promise<Pipeline[]> {
        const response = await fetch(`${API_BASE_URL}/pipelines`);
        if (!response.ok) throw new Error('Failed to fetch pipelines');
        return response.json();
    }

    async getPipeline(id: number): Promise<Pipeline> {
        const response = await fetch(`${API_BASE_URL}/pipelines/${id}`);
        if (!response.ok) throw new Error('Failed to fetch pipeline');
        return response.json();
    }

    async createPipeline(config: PipelineConfig): Promise<Pipeline> {
        const response = await fetch(`${API_BASE_URL}/pipelines`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        if (!response.ok) throw new Error('Failed to create pipeline');
        return response.json();
    }

    async getJobs(): Promise<Job[]> {
        const response = await fetch(`${API_BASE_URL}/jobs`);
        if (!response.ok) throw new Error('Failed to fetch jobs');
        return response.json();
    }

    async getJobsByPipeline(pipelineId: number): Promise<Job[]> {
        const response = await fetch(`${API_BASE_URL}/pipelines/${pipelineId}/jobs`);
        if (!response.ok) throw new Error('Failed to fetch pipeline jobs');
        return response.json();
    }

    async getJobDetails(jobId: number): Promise<JobWithDetails> {
        const response = await fetch(`${API_BASE_URL}/jobs/${jobId}/details`);
        if (!response.ok) throw new Error('Failed to fetch job details');
        return response.json();
    }

    async cancelJob(jobId: number): Promise<void> {
        const response = await fetch(`${API_BASE_URL}/jobs/${jobId}/cancel`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        if (!response.ok) throw new Error('Failed to cancel job');
    }

    async retryJob(jobId: number): Promise<Job> {
        const response = await fetch(`${API_BASE_URL}/jobs/${jobId}/retry`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        if (!response.ok) throw new Error('Failed to retry job');
        return response.json();
    }

    async streamJobLogs(jobId: number, follow: boolean = false): Promise<Response> {
        const url = `${API_BASE_URL}/jobs/${jobId}/logs/stream${follow ? '?follow=true' : ''}`;
        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to stream job logs');
        return response;
    }

    async getJobLogs(jobId: number): Promise<any> {
        const response = await fetch(`${API_BASE_URL}/jobs/${jobId}/logs`);
        if (!response.ok) throw new Error('Failed to fetch job logs');
        return response.json();
    }

    async createJob(pipelineId: number): Promise<Job> {
        const response = await fetch(`${API_BASE_URL}/pipelines/${pipelineId}/jobs`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({})
        });
        if (!response.ok) throw new Error('Failed to create job');
        return response.json();
    }
}

// Use real API service for production
export const apiService = new ApiService();
