import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { RefreshCw, Calendar, Eye, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@/components/ui/select';
import { apiService, Job, Pipeline } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function History() {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [pipelines, setPipelines] = useState<Pipeline[]>([]);
    const [filteredJobs, setFilteredJobs] = useState<Job[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchTerm, setSearchTerm] = useState('');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [pipelineFilter, setPipelineFilter] = useState<string>('all');
    const navigate = useNavigate();
    const { toast } = useToast();

    useEffect(() => {
        loadData();
    }, []);

    useEffect(() => {
        filterJobs();
    }, [jobs, searchTerm, statusFilter, pipelineFilter]);

    const loadData = async () => {
        try {
            const [jobsData, pipelinesData] = await Promise.all([
                apiService.getJobs(),
                apiService.getPipelines()
            ]);

            const sortedJobs = (jobsData || []).sort(
                (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
            );

            setJobs(sortedJobs);
            setPipelines(pipelinesData || []);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load job history",
                variant: "destructive"
            });
        } finally {
            setLoading(false);
        }
    };

    const filterJobs = () => {
        let filtered = [...jobs];

        if (searchTerm) {
            filtered = filtered.filter(job =>
                job.id.toString().includes(searchTerm) ||
                (job.branch && job.branch.toLowerCase().includes(searchTerm.toLowerCase())) ||
                (job.language && job.language.toLowerCase().includes(searchTerm.toLowerCase()))
            );
        }

        if (statusFilter !== 'all') {
            filtered = filtered.filter(job => job.status === statusFilter);
        }

        if (pipelineFilter !== 'all') {
            filtered = filtered.filter(job => job.pipeline_id.toString() === pipelineFilter);
        }

        setFilteredJobs(filtered);
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

    const getPipelineName = (pipelineId: number) => {
        const pipeline = pipelines.find(p => p.id === pipelineId);
        return pipeline?.name || `Pipeline ${pipelineId}`;
    };

    if (loading) {
        return (
            <div className="space-y-6">
                <h1 className="text-3xl font-bold">Job History</h1>
                <div className="grid gap-4">
                    {[1, 2, 3].map((i) => (
                        <Card key={i} className="p-6">
                            <div className="animate-pulse space-y-4">
                                <div className="h-4 bg-muted rounded w-1/4"></div>
                                <div className="h-3 bg-muted rounded w-3/4"></div>
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
                    <h1 className="text-3xl font-bold text-foreground">Job History</h1>
                    <p className="text-muted-foreground mt-1">
                        Complete execution history with filtering capabilities
                    </p>
                </div>
                <Button onClick={loadData} variant="outline">
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Refresh
                </Button>
            </div>

            <Card className="p-4">
                <div className="flex flex-col md:flex-row gap-4 items-start md:items-end">
                    <div className="flex-1">
                        <label className="text-sm font-medium mb-2 block">Search</label>
                        <div className="relative">
                            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <Input
                                placeholder="Search by job ID, branch, language..."
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                className="pl-10"
                            />
                        </div>
                    </div>

                    <div className="w-full md:w-40">
                        <label className="text-sm font-medium mb-2 block">Status</label>
                        <Select value={statusFilter} onValueChange={setStatusFilter}>
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">All Status</SelectItem>
                                <SelectItem value="completed">Completed</SelectItem>
                                <SelectItem value="running">Running</SelectItem>
                                <SelectItem value="failed">Failed</SelectItem>
                                <SelectItem value="cancelled">Cancelled</SelectItem>
                                <SelectItem value="pending">Pending</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="w-full md:w-48">
                        <label className="text-sm font-medium mb-2 block">Pipeline</label>
                        <Select value={pipelineFilter} onValueChange={setPipelineFilter}>
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">All Pipelines</SelectItem>
                                {pipelines.map(pipeline => (
                                    <SelectItem key={pipeline.id} value={pipeline.id.toString()}>
                                        {pipeline.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            </Card>

            <div className="text-sm text-muted-foreground">
                Showing {filteredJobs.length} of {jobs.length} jobs
            </div>

            {filteredJobs.length === 0 ? (
                <Card className="p-12 text-center">
                    <Calendar className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                    <h3 className="text-lg font-semibold mb-2">No jobs found</h3>
                    <p className="text-muted-foreground">
                        {jobs.length === 0 ? 'Run a pipeline to see job history here.' : 'Try adjusting your search criteria.'}
                    </p>
                </Card>
            ) : (
                <div className="space-y-3">
                    {filteredJobs.map((job) => (
                        <Card key={job.id} className="p-4 hover:shadow-md transition-shadow">
                            <div className="flex items-start justify-between">
                                <div className="flex-1">
                                    <div className="flex items-center gap-3 mb-3">
                                        <Badge className={getStatusColor(job.status)}>
                                            Job #{job.id}
                                        </Badge>
                                        <Badge variant="outline" className="text-xs">
                                            {getPipelineName(job.pipeline_id)}
                                        </Badge>
                                        {job.language && (
                                            <Badge variant="secondary" className="text-xs">
                                                {job.language}
                                            </Badge>
                                        )}
                                        {job.branch && (
                                            <Badge variant="outline" className="text-xs">
                                                {job.branch}
                                            </Badge>
                                        )}
                                    </div>

                                    <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm text-muted-foreground">
                                        <div>
                                            <span className="font-medium">Created:</span>
                                            <br />
                                            {new Date(job.created_at).toLocaleDateString()} {new Date(job.created_at).toLocaleTimeString()}
                                        </div>
                                        {job.started_at && (
                                            <div>
                                                <span className="font-medium">Started:</span>
                                                <br />
                                                {new Date(job.started_at).toLocaleDateString()} {new Date(job.started_at).toLocaleTimeString()}
                                            </div>
                                        )}
                                        <div>
                                            <span className="font-medium">Duration:</span>
                                            <br />
                                            {formatDuration(job.started_at || undefined, job.finished_at || undefined)}
                                        </div>
                                    </div>
                                </div>

                                <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => navigate(`/jobs/${job.id}`)}
                                >
                                    <Eye className="h-4 w-4 mr-1" />
                                    View
                                </Button>
                            </div>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    );
}
