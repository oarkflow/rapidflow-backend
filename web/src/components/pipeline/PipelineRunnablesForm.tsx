import { useState } from 'react';
import { Plus, Trash2, Container, Settings, Globe } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface Runnable {
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
}

interface PipelineRunnablesFormProps {
  runnables: Runnable[];
  onChange: (runnables: Runnable[]) => void;
}

const runnableTypes = [
  { value: 'docker_container', label: 'Docker Container', icon: Container },
  { value: 'kubernetes_deployment', label: 'Kubernetes Deployment', icon: Globe },
  { value: 'serverless_function', label: 'Serverless Function', icon: Settings }
];

const outputTypes = [
  { value: 'local', label: 'Local Storage' },
  { value: 'docker_registry', label: 'Docker Registry' },
  { value: 'artifact_storage', label: 'Artifact Storage' },
  { value: 'cloud_storage', label: 'Cloud Storage' }
];

export function PipelineRunnablesForm({ runnables, onChange }: PipelineRunnablesFormProps) {
  const [expandedRunnable, setExpandedRunnable] = useState<number | null>(null);

  const addRunnable = () => {
    const newRunnable: Runnable = {
      name: `runnable-${runnables.length + 1}`,
      type: 'docker_container',
      enabled: true,
      container_name: '',
      image_name: '',
      entrypoint: [],
      working_dir: '/workspace',
      ports: [],
      environment: {},
      config: {},
      outputs: []
    };
    onChange([...runnables, newRunnable]);
    setExpandedRunnable(runnables.length);
  };

  const removeRunnable = (index: number) => {
    const newRunnables = runnables.filter((_, i) => i !== index);
    onChange(newRunnables);
    if (expandedRunnable === index) {
      setExpandedRunnable(null);
    }
  };

  const updateRunnable = (index: number, field: keyof Runnable, value: any) => {
    const newRunnables = [...runnables];
    newRunnables[index] = { ...newRunnables[index], [field]: value };
    onChange(newRunnables);
  };

  const addOutput = (runnableIndex: number) => {
    const runnable = runnables[runnableIndex];
    const newOutput = { type: 'local', config: {} };
    const newOutputs = [...(runnable.outputs || []), newOutput];
    updateRunnable(runnableIndex, 'outputs', newOutputs);
  };

  const removeOutput = (runnableIndex: number, outputIndex: number) => {
    const runnable = runnables[runnableIndex];
    const newOutputs = (runnable.outputs || []).filter((_, i) => i !== outputIndex);
    updateRunnable(runnableIndex, 'outputs', newOutputs);
  };

  const updateOutput = (runnableIndex: number, outputIndex: number, field: string, value: any) => {
    const runnable = runnables[runnableIndex];
    const newOutputs = [...(runnable.outputs || [])];
    if (field === 'type') {
      newOutputs[outputIndex] = { ...newOutputs[outputIndex], type: value };
    } else {
      newOutputs[outputIndex] = {
        ...newOutputs[outputIndex],
        config: { ...newOutputs[outputIndex].config, [field]: value }
      };
    }
    updateRunnable(runnableIndex, 'outputs', newOutputs);
  };

  const addPort = (runnableIndex: number, port: string) => {
    const runnable = runnables[runnableIndex];
    const newPorts = [...(runnable.ports || []), port];
    updateRunnable(runnableIndex, 'ports', newPorts);
  };

  const removePort = (runnableIndex: number, portIndex: number) => {
    const runnable = runnables[runnableIndex];
    const newPorts = (runnable.ports || []).filter((_, i) => i !== portIndex);
    updateRunnable(runnableIndex, 'ports', newPorts);
  };

  const addEnvVar = (runnableIndex: number, key: string, value: string) => {
    const runnable = runnables[runnableIndex];
    const newEnv = { ...(runnable.environment || {}), [key]: value };
    updateRunnable(runnableIndex, 'environment', newEnv);
  };

  const removeEnvVar = (runnableIndex: number, key: string) => {
    const runnable = runnables[runnableIndex];
    const newEnv = { ...(runnable.environment || {}) };
    delete newEnv[key];
    updateRunnable(runnableIndex, 'environment', newEnv);
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Runnables & Deployments</h3>
            <p className="text-sm text-muted-foreground">
              Configure how to package and deploy your built application
            </p>
          </div>
          <Button onClick={addRunnable} className="bg-gradient-to-r from-primary to-primary-hover">
            <Plus className="h-4 w-4 mr-2" />
            Add Runnable
          </Button>
        </div>
      </div>

      {runnables.length === 0 ? (
        <Card className="p-8 text-center bg-muted/30">
          <Container className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
          <h4 className="font-medium mb-2">No runnables configured</h4>
          <p className="text-sm text-muted-foreground mb-4">
            Add runnables to define how your application should be packaged and deployed
          </p>
          <Button onClick={addRunnable} variant="outline">
            <Plus className="h-4 w-4 mr-2" />
            Add First Runnable
          </Button>
        </Card>
      ) : (
        <div className="space-y-4">
          {runnables.map((runnable, index) => (
            <Card key={index} className="p-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Container className="h-5 w-5 text-muted-foreground" />
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">Runnable {index + 1}</Badge>
                      <Badge variant={runnable.enabled ? "default" : "secondary"}>
                        {runnable.enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                      <Badge variant="outline">{runnable.type}</Badge>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={runnable.enabled}
                      onCheckedChange={(checked) => updateRunnable(index, 'enabled', checked)}
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setExpandedRunnable(expandedRunnable === index ? null : index)}
                    >
                      {expandedRunnable === index ? 'Collapse' : 'Configure'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeRunnable(index)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {expandedRunnable === index && (
                  <div className="space-y-4 border-t pt-4">
                    <Tabs defaultValue="basic" className="space-y-4">
                      <TabsList className="grid w-full grid-cols-4">
                        <TabsTrigger value="basic">Basic</TabsTrigger>
                        <TabsTrigger value="container">Container</TabsTrigger>
                        <TabsTrigger value="environment">Environment</TabsTrigger>
                        <TabsTrigger value="outputs">Outputs</TabsTrigger>
                      </TabsList>

                      <TabsContent value="basic" className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="space-y-2">
                            <Label>Name</Label>
                            <Input
                              value={runnable.name}
                              onChange={(e) => updateRunnable(index, 'name', e.target.value)}
                              placeholder="e.g., production-container"
                            />
                          </div>
                          <div className="space-y-2">
                            <Label>Type</Label>
                            <Select
                              value={runnable.type}
                              onValueChange={(value) => updateRunnable(index, 'type', value)}
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                {runnableTypes.map((type) => (
                                  <SelectItem key={type.value} value={type.value}>
                                    <div className="flex items-center gap-2">
                                      <type.icon className="h-4 w-4" />
                                      {type.label}
                                    </div>
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      </TabsContent>

                      <TabsContent value="container" className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="space-y-2">
                            <Label>Container Name</Label>
                            <Input
                              value={runnable.container_name || ''}
                              onChange={(e) => updateRunnable(index, 'container_name', e.target.value)}
                              placeholder="e.g., my-app-prod"
                            />
                          </div>
                          <div className="space-y-2">
                            <Label>Image Name</Label>
                            <Input
                              value={runnable.image_name || ''}
                              onChange={(e) => updateRunnable(index, 'image_name', e.target.value)}
                              placeholder="e.g., my-app:latest"
                            />
                          </div>
                          <div className="space-y-2">
                            <Label>Working Directory</Label>
                            <Input
                              value={runnable.working_dir || ''}
                              onChange={(e) => updateRunnable(index, 'working_dir', e.target.value)}
                              placeholder="/workspace"
                            />
                          </div>
                        </div>

                        <div className="space-y-2">
                          <Label>Port Mappings</Label>
                          <div className="space-y-2">
                            {(runnable.ports || []).map((port, portIndex) => (
                              <div key={portIndex} className="flex items-center gap-2">
                                <Input value={port} readOnly className="flex-1" />
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => removePort(index, portIndex)}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                            ))}
                            <div className="flex items-center gap-2">
                              <Input
                                placeholder="e.g., 8080:3000"
                                onKeyPress={(e) => {
                                  if (e.key === 'Enter') {
                                    const target = e.target as HTMLInputElement;
                                    if (target.value.trim()) {
                                      addPort(index, target.value.trim());
                                      target.value = '';
                                    }
                                  }
                                }}
                              />
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={(e) => {
                                  const input = e.currentTarget.previousElementSibling as HTMLInputElement;
                                  if (input.value.trim()) {
                                    addPort(index, input.value.trim());
                                    input.value = '';
                                  }
                                }}
                              >
                                Add
                              </Button>
                            </div>
                          </div>
                        </div>
                      </TabsContent>

                      <TabsContent value="environment" className="space-y-4">
                        <div className="space-y-2">
                          <Label>Environment Variables</Label>
                          <div className="space-y-2">
                            {Object.entries(runnable.environment || {}).map(([key, value]) => (
                              <div key={key} className="flex items-center gap-2">
                                <Input value={key} readOnly className="w-1/3" />
                                <Input value={value} readOnly className="flex-1" />
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => removeEnvVar(index, key)}
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                            ))}
                            <div className="flex items-center gap-2">
                              <Input placeholder="Key" className="w-1/3" id={`env-key-${index}`} />
                              <Input placeholder="Value" className="flex-1" id={`env-value-${index}`} />
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                  const keyInput = document.getElementById(`env-key-${index}`) as HTMLInputElement;
                                  const valueInput = document.getElementById(`env-value-${index}`) as HTMLInputElement;
                                  if (keyInput.value.trim() && valueInput.value.trim()) {
                                    addEnvVar(index, keyInput.value.trim(), valueInput.value.trim());
                                    keyInput.value = '';
                                    valueInput.value = '';
                                  }
                                }}
                              >
                                Add
                              </Button>
                            </div>
                          </div>
                        </div>
                      </TabsContent>

                      <TabsContent value="outputs" className="space-y-4">
                        <div className="flex items-center justify-between">
                          <Label>Deployment Outputs</Label>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => addOutput(index)}
                          >
                            <Plus className="h-4 w-4 mr-1" />
                            Add Output
                          </Button>
                        </div>
                        
                        <div className="space-y-3">
                          {(runnable.outputs || []).map((output, outputIndex) => (
                            <Card key={outputIndex} className="p-3">
                              <div className="flex items-center justify-between mb-3">
                                <Badge variant="outline">Output {outputIndex + 1}</Badge>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => removeOutput(index, outputIndex)}
                                  className="text-destructive"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                              <div className="space-y-2">
                                <Select
                                  value={output.type}
                                  onValueChange={(value) => updateOutput(index, outputIndex, 'type', value)}
                                >
                                  <SelectTrigger>
                                    <SelectValue />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {outputTypes.map((type) => (
                                      <SelectItem key={type.value} value={type.value}>
                                        {type.label}
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                                <Input
                                  placeholder="Configuration path or URL"
                                  value={output.config?.path || ''}
                                  onChange={(e) => updateOutput(index, outputIndex, 'path', e.target.value)}
                                />
                              </div>
                            </Card>
                          ))}
                        </div>
                      </TabsContent>
                    </Tabs>
                  </div>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}