import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Play, MoreHorizontal, Eye, RotateCcw, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuTrigger 
} from '@/components/ui/dropdown-menu';
import { apiService, Job, Pipeline } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function PipelineJobs() {
  const { pipelineId } = useParams<{ pipelineId: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [pipeline, setPipeline] = useState<Pipeline | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (pipelineId) {
      loadPipelineAndJobs();
    }
  }, [pipelineId]);

  const loadPipelineAndJobs = async () => {
    if (!pipelineId) return;
    
    try {
      const [pipelineData, jobsData] = await Promise.all([
        apiService.getPipeline(parseInt(pipelineId)),
        apiService.getJobsByPipeline(parseInt(pipelineId))
      ]);
      
      setPipeline(pipelineData);
      setJobs(jobsData);
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to load pipeline jobs",
        variant: "destructive"
      });
    } finally {
      setLoading(false);
    }
  };

  const runPipeline = async () => {
    if (!pipelineId) return;
    
    try {
      await apiService.createJob(parseInt(pipelineId));
      toast({
        title: "Pipeline Started",
        description: "New job has been queued for execution",
      });
      loadPipelineAndJobs();
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
          <Button variant="outline" size="sm" onClick={() => navigate('/')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Pipelines
          </Button>
          <div className="animate-pulse">
            <div className="h-8 bg-muted rounded w-64"></div>
          </div>
        </div>
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
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" onClick={() => navigate('/')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Pipelines
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-foreground">
              {pipeline?.name || 'Pipeline'} Jobs
            </h1>
            <p className="text-muted-foreground mt-1">
              Execution history and status for this pipeline
            </p>
          </div>
        </div>
        
        <Button 
          onClick={runPipeline}
          className="bg-gradient-to-r from-success to-success/90 hover:from-success/90 hover:to-success text-success-foreground"
        >
          <Play className="h-4 w-4 mr-2" />
          Run Pipeline
        </Button>
      </div>

      {jobs.length === 0 ? (
        <Card className="p-12 text-center bg-gradient-to-br from-card to-muted/20">
          <Play className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
          <h3 className="text-lg font-semibold mb-2">No jobs executed yet</h3>
          <p className="text-muted-foreground mb-6">
            This pipeline hasn't been run yet. Click "Run Pipeline" to execute it.
          </p>
          <Button 
            onClick={runPipeline}
            className="bg-gradient-to-r from-primary to-primary-hover"
          >
            <Play className="h-4 w-4 mr-2" />
            Run First Job
          </Button>
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
                    <Badge className={getStatusColor(job.status)}>
                      {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
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
                    {job.branch && (
                      <div className="flex items-center gap-1">
                        <span>Branch: {job.branch}</span>
                      </div>
                    )}
                    <div className="flex items-center gap-1">
                      <Clock className="h-4 w-4" />
                      {new Date(job.created_at).toLocaleDateString()}
                    </div>
                    <div className="flex items-center gap-1">
                      <span>Duration: {formatDuration(job.started_at || undefined, job.finished_at || undefined)}</span>
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
                  <Button
                    onClick={() => navigate(`/jobs/${job.id}`)}
                    size="sm"
                    variant="outline"
                  >
                    <Eye className="h-4 w-4 mr-1" />
                    View Details
                  </Button>
                  
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
                      {job.status === 'failed' && (
                        <DropdownMenuItem>
                          <RotateCcw className="h-4 w-4 mr-2" />
                          Retry Job
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