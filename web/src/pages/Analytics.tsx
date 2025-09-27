import { useState, useEffect } from 'react';
import { BarChart3, TrendingUp, Clock, CheckCircle, XCircle, Activity } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { apiService, Job, Pipeline } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function Analytics() {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [pipelines, setPipelines] = useState<Pipeline[]>([]);
    const [loading, setLoading] = useState(true);
    const [timeRange, setTimeRange] = useState('7d');
    const { toast } = useToast();

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        try {
            const [jobsData, pipelinesData] = await Promise.all([
                apiService.getJobs(),
                apiService.getPipelines()
            ]);

            setJobs(jobsData || []);
            setPipelines(pipelinesData || []);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load analytics data",
                variant: "destructive"
            });
        } finally {
            setLoading(false);
        }
    };

    const getJobStats = () => {
        const total = jobs.length;
        const completed = jobs.filter(j => j.status === 'completed').length;
        const failed = jobs.filter(j => j.status === 'failed').length;
        const running = jobs.filter(j => j.status === 'running').length;
        const pending = jobs.filter(j => j.status === 'pending').length;

        const successRate = total > 0 ? Math.round((completed / total) * 100) : 0;
        const failureRate = total > 0 ? Math.round((failed / total) * 100) : 0;

        return { total, completed, failed, running, pending, successRate, failureRate };
    };

    const getPipelineStats = () => {
        return pipelines.map(pipeline => {
            const pipelineJobs = jobs.filter(j => j.pipeline_id === pipeline.id);
            const completed = pipelineJobs.filter(j => j.status === 'completed').length;
            const failed = pipelineJobs.filter(j => j.status === 'failed').length;
            const successRate = pipelineJobs.length > 0 ? Math.round((completed / pipelineJobs.length) * 100) : 0;

            return {
                pipeline,
                totalJobs: pipelineJobs.length,
                completed,
                failed,
                successRate
            };
        });
    };

    const getAverageDuration = () => {
        const completedJobs = jobs.filter(j => j.status === 'completed' && j.started_at && j.finished_at);
        if (completedJobs.length === 0) return 'N/A';

        const totalDuration = completedJobs.reduce((acc, job) => {
            const start = new Date(job.started_at!);
            const end = new Date(job.finished_at!);
            return acc + (end.getTime() - start.getTime());
        }, 0);

        const avgMs = totalDuration / completedJobs.length;
        const avgSeconds = Math.floor(avgMs / 1000);

        if (avgSeconds < 60) return `${avgSeconds}s`;
        if (avgSeconds < 3600) return `${Math.floor(avgSeconds / 60)}m ${avgSeconds % 60}s`;
        return `${Math.floor(avgSeconds / 3600)}h ${Math.floor((avgSeconds % 3600) / 60)}m`;
    };

    const stats = getJobStats();
    const pipelineStats = getPipelineStats();
    const avgDuration = getAverageDuration();

    if (loading) {
        return (
            <div className="space-y-6">
                <h1 className="text-3xl font-bold">Analytics</h1>
                <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
                    {[1, 2, 3, 4].map((i) => (
                        <Card key={i} className="p-6">
                            <div className="animate-pulse space-y-4">
                                <div className="h-4 bg-muted rounded w-1/2"></div>
                                <div className="h-8 bg-muted rounded w-3/4"></div>
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
                    <h1 className="text-3xl font-bold text-foreground">Analytics</h1>
                    <p className="text-muted-foreground mt-1">
                        Performance metrics and insights for your CI/CD pipelines
                    </p>
                </div>

                <Select value={timeRange} onValueChange={setTimeRange}>
                    <SelectTrigger className="w-48">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="24h">Last 24 hours</SelectItem>
                        <SelectItem value="7d">Last 7 days</SelectItem>
                        <SelectItem value="30d">Last 30 days</SelectItem>
                        <SelectItem value="90d">Last 90 days</SelectItem>
                    </SelectContent>
                </Select>
            </div>

            {/* Overview Stats */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
                <Card className="p-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-primary/10 rounded-lg">
                            <BarChart3 className="h-5 w-5 text-primary" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Total Jobs</p>
                            <p className="text-2xl font-bold">{stats.total}</p>
                        </div>
                    </div>
                </Card>

                <Card className="p-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-success/10 rounded-lg">
                            <CheckCircle className="h-5 w-5 text-success" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Success Rate</p>
                            <p className="text-2xl font-bold">{stats.successRate}%</p>
                        </div>
                    </div>
                </Card>

                <Card className="p-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-destructive/10 rounded-lg">
                            <XCircle className="h-5 w-5 text-destructive" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Failure Rate</p>
                            <p className="text-2xl font-bold">{stats.failureRate}%</p>
                        </div>
                    </div>
                </Card>

                <Card className="p-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-warning/10 rounded-lg">
                            <Clock className="h-5 w-5 text-warning" />
                        </div>
                        <div>
                            <p className="text-sm text-muted-foreground">Avg Duration</p>
                            <p className="text-2xl font-bold">{avgDuration}</p>
                        </div>
                    </div>
                </Card>
            </div>

            {/* Status Breakdown */}
            <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Job Status Breakdown</h3>
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                    <div className="text-center">
                        <div className="flex items-center justify-center gap-2 mb-2">
                            <CheckCircle className="h-4 w-4 text-success" />
                            <span className="text-sm font-medium">Completed</span>
                        </div>
                        <Badge className="bg-success text-success-foreground">
                            {stats.completed}
                        </Badge>
                    </div>

                    <div className="text-center">
                        <div className="flex items-center justify-center gap-2 mb-2">
                            <XCircle className="h-4 w-4 text-destructive" />
                            <span className="text-sm font-medium">Failed</span>
                        </div>
                        <Badge className="bg-destructive text-destructive-foreground">
                            {stats.failed}
                        </Badge>
                    </div>

                    <div className="text-center">
                        <div className="flex items-center justify-center gap-2 mb-2">
                            <Activity className="h-4 w-4 text-primary" />
                            <span className="text-sm font-medium">Running</span>
                        </div>
                        <Badge className="bg-primary text-primary-foreground">
                            {stats.running}
                        </Badge>
                    </div>

                    <div className="text-center">
                        <div className="flex items-center justify-center gap-2 mb-2">
                            <Clock className="h-4 w-4 text-warning" />
                            <span className="text-sm font-medium">Pending</span>
                        </div>
                        <Badge className="bg-warning text-warning-foreground">
                            {stats.pending}
                        </Badge>
                    </div>

                    <div className="text-center">
                        <div className="flex items-center justify-center gap-2 mb-2">
                            <TrendingUp className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm font-medium">Total</span>
                        </div>
                        <Badge variant="outline">
                            {stats.total}
                        </Badge>
                    </div>
                </div>
            </Card>

            {/* Pipeline Performance */}
            <Card className="p-6">
                <h3 className="text-lg font-semibold mb-4">Pipeline Performance</h3>
                <div className="space-y-4">
                    {pipelineStats.length === 0 ? (
                        <p className="text-muted-foreground text-center py-8">
                            No pipeline data available yet.
                        </p>
                    ) : (
                        pipelineStats.map((stat) => (
                            <div key={stat.pipeline.id} className="flex items-center justify-between p-4 border rounded-lg">
                                <div className="flex-1">
                                    <h4 className="font-medium">{stat.pipeline.name}</h4>
                                    <p className="text-sm text-muted-foreground">
                                        {stat.totalJobs} total jobs • {stat.completed} completed • {stat.failed} failed
                                    </p>
                                </div>

                                <div className="flex items-center gap-4">
                                    <div className="text-right">
                                        <div className="text-sm text-muted-foreground">Success Rate</div>
                                        <div className={`font-bold ${stat.successRate >= 80 ? 'text-success' : stat.successRate >= 60 ? 'text-warning' : 'text-destructive'}`}>
                                            {stat.successRate}%
                                        </div>
                                    </div>

                                    <div className="w-24 bg-muted rounded-full h-2">
                                        <div
                                            className={`h-2 rounded-full ${stat.successRate >= 80 ? 'bg-success' : stat.successRate >= 60 ? 'bg-warning' : 'bg-destructive'}`}
                                            style={{ width: `${stat.successRate}%` }}
                                        ></div>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}
                </div>
            </Card>
        </div>
    );
}
