import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { RefreshCw, Play, Square, MoreHorizontal, Eye, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { apiService, Job } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function Jobs() {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [loading, setLoading] = useState(true);
    const navigate = useNavigate();
    const { toast } = useToast();

    useEffect(() => {
        loadJobs();
    }, []);

    const cancelJob = async (jobId: number) => {
        try {
            await apiService.cancelJob(jobId);
            toast({
                title: "Job Cancelled",
                description: `Job #${jobId} has been cancelled`,
            });
            loadJobs(); // Refresh the jobs list
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to cancel job",
                variant: "destructive"
            });
        }
    };

    const retryJob = async (jobId: number) => {
        try {
            await apiService.retryJob(jobId);
            toast({
                title: "Job Retried",
                description: `Job #${jobId} has been retried`,
            });
            loadJobs(); // Refresh the jobs list
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to retry job",
                variant: "destructive"
            });
        }
    };

    const loadJobs = async () => {
        try {
            const data = await apiService.getJobs();
            setJobs(data || []);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load jobs",
                variant: "destructive"
            });
        } finally {
            setLoading(false);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'completed': return 'bg-success text-success-foreground';
            case 'running': return 'bg-primary text-primary-foreground animate-pulse';
            case 'failed': return 'bg-destructive text-destructive-foreground';
            case 'cancelled': return 'bg-muted text-muted-foreground';
            case 'stopped': return 'bg-muted text-muted-foreground';
            case 'pending': return 'bg-warning text-warning-foreground';
            default: return 'bg-muted text-muted-foreground';
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'running': return <Play className="h-3 w-3" />;
            case 'completed': return <span className="h-3 w-3">✓</span>;
            case 'failed': return <span className="h-3 w-3">✗</span>;
            case 'cancelled':
            case 'stopped': return <Square className="h-3 w-3" />;
            default: return <span className="h-3 w-3">●</span>;
        }
    };

    const formatDuration = (startedAt?: string, finishedAt?: string) => {
        if (!startedAt) return 'Not started';
        const start = new Date(startedAt);
        const end = finishedAt ? new Date(finishedAt) : new Date();
        const duration = Math.floor((end.getTime() - start.getTime()) / 1000);

        if (duration < 60) return `${duration}s`;
        if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
        return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
    };

    if (loading) {
        return (
            <div className="space-y-6">
                <div className="flex items-center justify-between">
                    <h1 className="text-3xl font-bold">Pipeline Jobs</h1>
                </div>
                <div className="grid gap-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="p-6">
                            <div className="animate-pulse space-y-4">
                                <div className="h-4 bg-muted rounded w-1/4"></div>
                                <div className="h-3 bg-muted rounded w-3/4"></div>
                                <div className="flex gap-2">
                                    <div className="h-6 bg-muted rounded w-16"></div>
                                    <div className="h-6 bg-muted rounded w-16"></div>
                                </div>
                            </div>
                        </Card>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-foreground">Pipeline Jobs</h1>
                    <p className="text-muted-foreground mt-1">
                        Monitor and manage pipeline execution history
                    </p>
                </div>
                <Button onClick={loadJobs} variant="outline">
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Refresh
                </Button>
            </div>

            {jobs.length === 0 ? (
                <Card className="p-12 text-center bg-gradient-to-br from-card to-muted/20">
                    <Play className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                    <h3 className="text-lg font-semibold mb-2">No jobs executed yet</h3>
                    <p className="text-muted-foreground mb-6">
                        Run a pipeline to see job execution history here.
                    </p>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {jobs.map((job) => (
                        <Card key={job.id} className="p-6 hover:shadow-lg transition-shadow bg-gradient-to-r from-card to-card/80">
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-2">
                                        <h3 className="text-xl font-semibold text-foreground">
                                            Job #{job.id}
                                        </h3>
                                        <Badge variant="outline" className="text-xs">
                                            Pipeline {job.pipeline_id}
                                        </Badge>
                                        {job.language && (
                                            <Badge variant="secondary" className="text-xs">
                                                {job.language}
                                            </Badge>
                                        )}
                                        {job.version && (
                                            <Badge variant="outline" className="text-xs">
                                                v{job.version}
                                            </Badge>
                                        )}
                                    </div>

                                    <div className="flex items-center gap-4 text-sm text-muted-foreground mb-4">
                                        <div className="flex items-center gap-1">
                                            <Badge
                                                className={`${getStatusColor(job.status)} flex items-center gap-1`}
                                            >
                                                {getStatusIcon(job.status)}
                                                {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
                                            </Badge>
                                        </div>
                                        {job.branch && (
                                            <div className="flex items-center gap-1">
                                                <span>Branch: {job.branch}</span>
                                            </div>
                                        )}
                                        <div className="flex items-center gap-1">
                                            <span>Created: {new Date(job.created_at).toLocaleDateString()}</span>
                                        </div>
                                        <div className="flex items-center gap-1">
                                            <span>Duration: {formatDuration(job.started_at, job.finished_at)}</span>
                                        </div>
                                    </div>

                                    {(job.started_at || job.finished_at) && (
                                        <div className="flex gap-4 text-xs text-muted-foreground">
                                            {job.started_at && (
                                                <span>Started: {new Date(job.started_at).toLocaleString()}</span>
                                            )}
                                            {job.finished_at && (
                                                <span>Finished: {new Date(job.finished_at).toLocaleString()}</span>
                                            )}
                                        </div>
                                    )}
                                </div>

                                <div className="flex items-center gap-2">
                                    {job.status === 'failed' && (
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            className="border-warning text-warning hover:bg-warning hover:text-warning-foreground"
                                            onClick={() => retryJob(job.id)}
                                        >
                                            <RotateCcw className="h-4 w-4 mr-1" />
                                            Retry
                                        </Button>
                                    )}

                                    {job.status === 'running' && (
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            className="border-destructive text-destructive hover:bg-destructive hover:text-destructive-foreground"
                                            onClick={() => cancelJob(job.id)}
                                        >
                                            <Square className="h-4 w-4 mr-1" />
                                            Cancel
                                        </Button>
                                    )}

                                    <DropdownMenu>
                                        <DropdownMenuTrigger asChild>
                                            <Button variant="outline" size="sm">
                                                <MoreHorizontal className="h-4 w-4" />
                                            </Button>
                                        </DropdownMenuTrigger>
                                        <DropdownMenuContent align="end">
                                            <DropdownMenuItem onClick={() => navigate(`/jobs/${job.id}`)}>
                                                <Eye className="h-4 w-4 mr-2" />
                                                View Details
                                            </DropdownMenuItem>
                                            <DropdownMenuItem>
                                                <RefreshCw className="h-4 w-4 mr-2" />
                                                View Logs
                                            </DropdownMenuItem>
                                            {job.status === 'failed' && (
                                                <DropdownMenuItem onClick={() => retryJob(job.id)}>
                                                    <RotateCcw className="h-4 w-4 mr-2" />
                                                    Retry Job
                                                </DropdownMenuItem>
                                            )}
                                            {job.status === 'running' && (
                                                <DropdownMenuItem onClick={() => cancelJob(job.id)}>
                                                    <Square className="h-4 w-4 mr-2" />
                                                    Cancel Job
                                                </DropdownMenuItem>
                                            )}
                                        </DropdownMenuContent>
                                    </DropdownMenu>
                                </div>
                            </div>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    );
}
