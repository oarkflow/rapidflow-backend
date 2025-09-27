import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, RefreshCw, Play, Square, Download, Terminal, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { apiService, JobWithDetails } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function JobDetails() {
    const { jobId } = useParams<{ jobId: string }>();
    const navigate = useNavigate();
    const { toast } = useToast();
    const [jobDetails, setJobDetails] = useState<JobWithDetails | null>(null);
    const [logs, setLogs] = useState<any>(null);
    const [loading, setLoading] = useState(true);
    const [logsLoading, setLogsLoading] = useState(false);

    useEffect(() => {
        if (jobId) {
            loadJobDetails();
            loadLogs();

            // Poll logs every 2 seconds for running jobs
            const interval = setInterval(() => {
                if (jobDetails?.job.status === 'running') {
                    loadLogs();
                }
            }, 2000);

            return () => clearInterval(interval);
        }
    }, [jobId, jobDetails?.job.status]);

    const cancelJob = async () => {
        if (!jobId) return;

        try {
            await apiService.cancelJob(parseInt(jobId));
            toast({
                title: "Job Cancelled",
                description: `Job #${jobId} has been cancelled`,
            });
            loadJobDetails(); // Refresh the job details
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to cancel job",
                variant: "destructive"
            });
        }
    };

    const retryJob = async () => {
        if (!jobId) return;

        try {
            const newJob = await apiService.retryJob(parseInt(jobId));
            toast({
                title: "Job Retried",
                description: `New job #${newJob.id} has been created`,
            });
            // Navigate to the new job
            navigate(`/jobs/${newJob.id}`);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to retry job",
                variant: "destructive"
            });
        }
    };

    const loadJobDetails = async () => {
        if (!jobId) return;

        try {
            const data = await apiService.getJobDetails(parseInt(jobId));
            setJobDetails(data);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load job details",
                variant: "destructive"
            });
        } finally {
            setLoading(false);
        }
    };

    const loadLogs = async () => {
        if (!jobId) return;

        setLogsLoading(true);
        try {
            const logsData = await apiService.getJobLogs(parseInt(jobId));
            setLogs(logsData);
        } catch (error) {
            console.error('Failed to load logs:', error);
        } finally {
            setLogsLoading(false);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'completed': return 'bg-success text-success-foreground';
            case 'running': return 'bg-primary text-primary-foreground animate-pulse';
            case 'failed': return 'bg-destructive text-destructive-foreground';
            case 'cancelled':
            case 'stopped': return 'bg-muted text-muted-foreground';
            case 'pending': return 'bg-warning text-warning-foreground';
            default: return 'bg-muted text-muted-foreground';
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
                <div className="flex items-center gap-4">
                    <Button variant="outline" size="sm" onClick={() => navigate('/jobs')}>
                        <ArrowLeft className="h-4 w-4 mr-2" />
                        Back to Jobs
                    </Button>
                    <div className="animate-pulse">
                        <div className="h-8 bg-muted rounded w-48"></div>
                    </div>
                </div>
                <Card className="p-6">
                    <div className="animate-pulse space-y-4">
                        <div className="h-4 bg-muted rounded w-1/4"></div>
                        <div className="h-3 bg-muted rounded w-3/4"></div>
                    </div>
                </Card>
            </div>
        );
    }

    if (!jobDetails) {
        return (
            <div className="space-y-6">
                <div className="flex items-center gap-4">
                    <Button variant="outline" size="sm" onClick={() => navigate('/jobs')}>
                        <ArrowLeft className="h-4 w-4 mr-2" />
                        Back to Jobs
                    </Button>
                </div>
                <Card className="p-12 text-center">
                    <h3 className="text-lg font-semibold mb-2">Job not found</h3>
                    <p className="text-muted-foreground">The requested job could not be found.</p>
                </Card>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="outline" size="sm" onClick={() => navigate('/jobs')}>
                        <ArrowLeft className="h-4 w-4 mr-2" />
                        Back to Jobs
                    </Button>
                    <div>
                        <h1 className="text-3xl font-bold text-foreground">Job #{jobDetails.job.id}</h1>
                        <p className="text-muted-foreground">Pipeline: {jobDetails.pipeline.name}</p>
                    </div>
                </div>

                <div className="flex items-center gap-2">
                    {jobDetails.job.status === 'failed' && (
                        <Button
                            variant="outline"
                            className="border-warning text-warning hover:bg-warning hover:text-warning-foreground"
                            onClick={retryJob}
                        >
                            <RotateCcw className="h-4 w-4 mr-1" />
                            Retry
                        </Button>
                    )}
                    {jobDetails.job.status === 'running' && (
                        <Button
                            variant="outline"
                            className="border-destructive text-destructive hover:bg-destructive hover:text-destructive-foreground"
                            onClick={cancelJob}
                        >
                            <Square className="h-4 w-4 mr-1" />
                            Cancel
                        </Button>
                    )}
                    <Button onClick={loadLogs} variant="outline" disabled={logsLoading}>
                        <RefreshCw className={`h-4 w-4 mr-2 ${logsLoading ? 'animate-spin' : ''}`} />
                        Refresh
                    </Button>
                </div>
            </div>

            {/* Job Status Card */}
            <Card className="p-6">
                <div className="flex items-start justify-between mb-4">
                    <div className="space-y-2">
                        <div className="flex items-center gap-3">
                            <Badge className={getStatusColor(jobDetails.job.status)}>
                                {jobDetails.job.status.charAt(0).toUpperCase() + jobDetails.job.status.slice(1)}
                            </Badge>
                            {jobDetails.job.language && (
                                <Badge variant="outline">{jobDetails.job.language}</Badge>
                            )}
                            {jobDetails.job.version && (
                                <Badge variant="secondary">v{jobDetails.job.version}</Badge>
                            )}
                        </div>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                            <div>
                                <span className="text-muted-foreground">Created:</span>
                                <br />
                                <span>{new Date(jobDetails.job.created_at).toLocaleString()}</span>
                            </div>
                            {jobDetails.job.started_at && (
                                <div>
                                    <span className="text-muted-foreground">Started:</span>
                                    <br />
                                    <span>{new Date(jobDetails.job.started_at).toLocaleString()}</span>
                                </div>
                            )}
                            {jobDetails.job.finished_at && (
                                <div>
                                    <span className="text-muted-foreground">Finished:</span>
                                    <br />
                                    <span>{new Date(jobDetails.job.finished_at).toLocaleString()}</span>
                                </div>
                            )}
                            <div>
                                <span className="text-muted-foreground">Duration:</span>
                                <br />
                                <span>{formatDuration(jobDetails.job.started_at || undefined, jobDetails.job.finished_at || undefined)}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </Card>

            {/* Tabs for different views */}
            <Tabs defaultValue="logs" className="space-y-4">
                <TabsList>
                    <TabsTrigger value="logs">Logs</TabsTrigger>
                    <TabsTrigger value="steps">Steps</TabsTrigger>
                    <TabsTrigger value="environment">Environment</TabsTrigger>
                    <TabsTrigger value="runnables">Runnables</TabsTrigger>
                </TabsList>

                <TabsContent value="logs">
                    <Card className="p-6">
                        <div className="flex items-center justify-between mb-4">
                            <div className="flex items-center gap-2">
                                <Terminal className="h-5 w-5" />
                                <h3 className="font-semibold">Job Logs</h3>
                                {jobDetails.job.status === 'running' && (
                                    <Badge variant="outline" className="animate-pulse">Live</Badge>
                                )}
                            </div>
                            <Button variant="outline" size="sm">
                                <Download className="h-4 w-4 mr-2" />
                                Download
                            </Button>
                        </div>

                        <ScrollArea className="h-96 border rounded bg-black text-green-400 p-4 font-mono text-sm">
                            {logs && logs.logs && logs.logs.length > 0 ? (
                                <div className="space-y-2">
                                    {logs.logs.map((logEntry: any, index: number) => (
                                        <div key={index} className="border-b border-gray-700 pb-2">
                                            <div className="text-yellow-400 text-xs mb-1">
                                                Step {logEntry.order_num}: {logEntry.type}
                                            </div>
                                            <div className="text-gray-300 text-xs mb-1">
                                                Status: <span className={`${logEntry.status === 'success' ? 'text-green-400' : logEntry.status === 'failed' ? 'text-red-400' : 'text-yellow-400'}`}>
                                                    {logEntry.status}
                                                </span>
                                            </div>
                                            <pre className="whitespace-pre-wrap text-green-400">
                                                {logEntry.output || 'No output'}
                                            </pre>
                                        </div>
                                    ))}
                                    {jobDetails.job.status === 'running' && logsLoading && (
                                        <div className="text-yellow-400 animate-pulse">
                                            Loading more logs...
                                        </div>
                                    )}
                                </div>
                            ) : (
                                <div className="text-gray-400">
                                    No logs available yet. Check back when the job starts running.
                                </div>
                            )}
                        </ScrollArea>
                    </Card>
                </TabsContent>

                <TabsContent value="steps">
                    <div className="space-y-4">
                        {(jobDetails.steps || []).map((step, index) => (
                            <Card key={step.id} className="p-4">
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-3 mb-2">
                                            <Badge variant="outline">Step {step.order_num}</Badge>
                                            <Badge variant="secondary">{step.type}</Badge>
                                            <Badge className={getStatusColor(step.status)}>
                                                {step.status}
                                            </Badge>
                                        </div>
                                        <pre className="bg-muted p-3 rounded text-sm overflow-x-auto">
                                            {step.content}
                                        </pre>
                                        {step.output && (
                                            <div className="mt-3">
                                                <h4 className="font-medium text-sm mb-2">Output:</h4>
                                                <pre className="bg-black text-green-400 p-3 rounded text-sm overflow-x-auto">
                                                    {step.output}
                                                </pre>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </Card>
                        ))}
                    </div>
                </TabsContent>

                <TabsContent value="environment">
                    <Card className="p-6">
                        <h3 className="font-semibold mb-4">Environment Variables</h3>
                        {(jobDetails.environments || []).length > 0 ? (
                            <div className="space-y-2">
                                {jobDetails.environments.map((env) => (
                                    <div key={env.id} className="flex items-center justify-between py-2 border-b">
                                        <span className="font-mono text-sm">{env.key}</span>
                                        <span className="text-muted-foreground font-mono text-sm">{env.value}</span>
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="text-muted-foreground">No environment variables configured.</p>
                        )}
                    </Card>
                </TabsContent>

                <TabsContent value="runnables">
                    <Card className="p-6">
                        <h3 className="font-semibold mb-4">Runnables & Deployments</h3>
                        {(jobDetails.runnables || []).length > 0 ? (
                            <div className="space-y-4">
                                {jobDetails.runnables.map((runnable) => (
                                    <div key={runnable.id} className="border rounded p-4">
                                        <div className="flex items-center gap-2 mb-2">
                                            <Badge variant="outline">{runnable.name}</Badge>
                                            <Badge variant="secondary">{runnable.type}</Badge>
                                            <Badge className={getStatusColor(runnable.status)}>
                                                {runnable.status}
                                            </Badge>
                                        </div>
                                        {runnable.output && (
                                            <pre className="bg-muted p-3 rounded text-sm overflow-x-auto">
                                                {runnable.output}
                                            </pre>
                                        )}
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <p className="text-muted-foreground">No runnables configured for this job.</p>
                        )}
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    );
}
