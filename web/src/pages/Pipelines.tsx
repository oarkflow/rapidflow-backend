import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, GitBranch, Clock, Play, MoreHorizontal, Settings, Trash2, Activity } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { apiService, Pipeline, PipelineConfig } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function Pipelines() {
    const [pipelines, setPipelines] = useState<Pipeline[]>([]);
    const [loading, setLoading] = useState(true);
    const navigate = useNavigate();
    const { toast } = useToast();

    useEffect(() => {
        loadPipelines();
    }, []);

    const loadPipelines = async () => {
        try {
            const data = await apiService.getPipelines();
            setPipelines(data || []);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to load pipelines",
                variant: "destructive"
            });
        } finally {
            setLoading(false);
        }
    };

    const runPipeline = async (pipeline: Pipeline) => {
        try {
            await apiService.createJob(pipeline.id);
            toast({
                title: "Pipeline Started",
                description: `${pipeline.name} has been queued for execution`,
            });
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to start pipeline",
                variant: "destructive"
            });
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'completed': return 'bg-success text-success-foreground';
            case 'running': return 'bg-primary text-primary-foreground';
            case 'failed': return 'bg-destructive text-destructive-foreground';
            case 'pending': return 'bg-warning text-warning-foreground';
            default: return 'bg-muted text-muted-foreground';
        }
    };

    if (loading) {
        return (
            <div className="space-y-6">
                <div className="flex items-center justify-between">
                    <h1 className="text-3xl font-bold">Pipelines</h1>
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
                    <h1 className="text-3xl font-bold text-foreground">Pipelines</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage and execute your CI/CD pipelines
                    </p>
                </div>
                <Button
                    onClick={() => navigate('/pipelines/new')}
                    className="bg-gradient-to-r from-primary to-primary-hover hover:from-primary-hover hover:to-primary"
                >
                    <Plus className="h-4 w-4 mr-2" />
                    Create Pipeline
                </Button>
            </div>

            {pipelines.length === 0 ? (
                <Card className="p-12 text-center bg-gradient-to-br from-card to-muted/20">
                    <GitBranch className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                    <h3 className="text-lg font-semibold mb-2">No pipelines yet</h3>
                    <p className="text-muted-foreground mb-6">
                        Create your first pipeline to get started with automated builds and deployments.
                    </p>
                    <Button
                        onClick={() => navigate('/pipelines/new')}
                        className="bg-gradient-to-r from-primary to-primary-hover"
                    >
                        <Plus className="h-4 w-4 mr-2" />
                        Create Your First Pipeline
                    </Button>
                </Card>
            ) : (
                <div className="grid gap-4">
                    {pipelines.map((pipeline) => {
                        const config: PipelineConfig = JSON.parse(pipeline.config);
                        return (
                            <Card key={pipeline.id} className="p-6 hover:shadow-lg transition-shadow bg-gradient-to-r from-card to-card/80">
                                <div className="flex items-start justify-between">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-3 mb-2">
                                            <h3 className="text-xl font-semibold text-foreground">{pipeline.name}</h3>
                                            <Badge variant="outline" className="text-xs">
                                                {config.language || 'Generic'}
                                            </Badge>
                                            {config.version && (
                                                <Badge variant="secondary" className="text-xs">
                                                    v{config.version}
                                                </Badge>
                                            )}
                                        </div>

                                        <div className="flex items-center gap-4 text-sm text-muted-foreground mb-4">
                                            <div className="flex items-center gap-1">
                                                <GitBranch className="h-4 w-4" />
                                                {config.branch || 'main'}
                                            </div>
                                            <div className="flex items-center gap-1">
                                                <Clock className="h-4 w-4" />
                                                {new Date(pipeline.created_at).toLocaleDateString()}
                                            </div>
                                            <div className="flex items-center gap-1">
                                                <span>{config.steps?.length || 0} steps</span>
                                            </div>
                                        </div>

                                        <div className="flex gap-2">
                                            <Badge
                                                variant="secondary"
                                                className={getStatusColor('completed')}
                                            >
                                                Last run: Completed
                                            </Badge>
                                        </div>
                                    </div>

                                    <div className="flex items-center gap-2">
                                        <Button
                                            onClick={() => runPipeline(pipeline)}
                                            size="sm"
                                            className="bg-gradient-to-r from-success to-success/90 hover:from-success/90 hover:to-success text-success-foreground"
                                        >
                                            <Play className="h-4 w-4 mr-1" />
                                            Run
                                        </Button>

                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button variant="outline" size="sm">
                                                    <MoreHorizontal className="h-4 w-4" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end">
                                                <DropdownMenuItem onClick={() => navigate(`/pipelines/${pipeline.id}/jobs`)}>
                                                    <Activity className="h-4 w-4 mr-2" />
                                                    View Jobs
                                                </DropdownMenuItem>
                                                <DropdownMenuItem onClick={() => navigate(`/pipelines/${pipeline.id}/edit`)}>
                                                    <Settings className="h-4 w-4 mr-2" />
                                                    Edit
                                                </DropdownMenuItem>
                                                <DropdownMenuItem className="text-destructive">
                                                    <Trash2 className="h-4 w-4 mr-2" />
                                                    Delete
                                                </DropdownMenuItem>
                                            </DropdownMenuContent>
                                        </DropdownMenu>
                                    </div>
                                </div>
                            </Card>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
