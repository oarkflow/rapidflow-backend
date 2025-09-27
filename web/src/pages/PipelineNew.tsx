import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Save, Play } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PipelineBasicForm } from '@/components/pipeline/PipelineBasicForm';
import { PipelineStepsForm } from '@/components/pipeline/PipelineStepsForm';
import { PipelineEnvironmentForm } from '@/components/pipeline/PipelineEnvironmentForm';
import { PipelineFilesForm } from '@/components/pipeline/PipelineFilesForm';
import { PipelineProvidersForm } from '@/components/pipeline/PipelineProvidersForm';
import { PipelineRunnablesForm } from '@/components/pipeline/PipelineRunnablesForm';
import { apiService, PipelineConfig } from '@/services/api';
import { useToast } from '@/hooks/use-toast';

export default function PipelineNew() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<PipelineConfig>({
    name: '',
    language: 'nodejs',
    version: '18',
    branch: 'main',
    folder: './',
    expose_ports: false,
    temporary: false,
    env: {},
    steps: [
      {
        type: 'bash',
        content: 'echo "Hello World"',
        files: {}
      }
    ],
    runnables: []
  });

  const updateConfig = (updates: Partial<PipelineConfig>) => {
    setConfig(prev => ({ ...prev, ...updates }));
  };

  const savePipeline = async (runAfterSave = false) => {
    if (!config.name.trim()) {
      toast({
        title: "Validation Error",
        description: "Pipeline name is required",
        variant: "destructive"
      });
      return;
    }

    setSaving(true);
    try {
      const pipeline = await apiService.createPipeline(config);
      toast({
        title: "Success",
        description: `Pipeline "${pipeline.name}" created successfully`,
      });

      if (runAfterSave) {
        await apiService.createJob(pipeline.id);
        toast({
          title: "Pipeline Started",
          description: "Pipeline has been queued for execution",
        });
      }

      navigate('/');
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to create pipeline",
        variant: "destructive"
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button 
            variant="outline" 
            size="sm" 
            onClick={() => navigate('/')}
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-foreground">Create Pipeline</h1>
            <p className="text-muted-foreground mt-1">
              Configure your CI/CD pipeline with steps, environment, and deployment options
            </p>
          </div>
        </div>
        
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => savePipeline(false)}
            disabled={saving}
          >
            <Save className="h-4 w-4 mr-2" />
            {saving ? 'Saving...' : 'Save'}
          </Button>
          <Button
            onClick={() => savePipeline(true)}
            disabled={saving}
            className="bg-gradient-to-r from-success to-success/90 hover:from-success/90 hover:to-success text-success-foreground"
          >
            <Play className="h-4 w-4 mr-2" />
            Save & Run
          </Button>
        </div>
      </div>

      <Card className="p-6">
        <Tabs defaultValue="basic" className="space-y-6">
          <TabsList className="grid w-full grid-cols-6">
            <TabsTrigger value="basic">Basic</TabsTrigger>
            <TabsTrigger value="steps">Steps</TabsTrigger>
            <TabsTrigger value="environment">Environment</TabsTrigger>
            <TabsTrigger value="files">Files</TabsTrigger>
            <TabsTrigger value="runnables">Runnables</TabsTrigger>
            <TabsTrigger value="providers">Providers</TabsTrigger>
          </TabsList>

          <TabsContent value="basic">
            <PipelineBasicForm 
              config={config}
              onChange={updateConfig}
            />
          </TabsContent>

          <TabsContent value="steps">
            <PipelineStepsForm 
              steps={config.steps}
              onChange={(steps) => updateConfig({ steps })}
            />
          </TabsContent>

          <TabsContent value="environment">
            <PipelineEnvironmentForm 
              env={config.env || {}}
              onChange={(env) => updateConfig({ env })}
            />
          </TabsContent>

          <TabsContent value="files">
            <PipelineFilesForm 
              steps={config.steps}
              onChange={(steps) => updateConfig({ steps })}
            />
          </TabsContent>

          <TabsContent value="runnables">
            <PipelineRunnablesForm 
              runnables={config.runnables || []}
              onChange={(runnables) => updateConfig({ runnables })}
            />
          </TabsContent>

          <TabsContent value="providers">
            <PipelineProvidersForm 
              config={config}
              onChange={updateConfig}
            />
          </TabsContent>
        </Tabs>
      </Card>
    </div>
  );
}