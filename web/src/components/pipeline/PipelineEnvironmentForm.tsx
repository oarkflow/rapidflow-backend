import { useState } from 'react';
import { Plus, Trash2, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';

interface PipelineEnvironmentFormProps {
  env: Record<string, string>;
  onChange: (env: Record<string, string>) => void;
}

interface EnvVar {
  key: string;
  value: string;
  isSecret: boolean;
}

const commonEnvVars = [
  { key: 'NODE_ENV', value: 'production', description: 'Node.js environment' },
  { key: 'PORT', value: '3000', description: 'Application port' },
  { key: 'DATABASE_URL', value: 'postgresql://user:pass@localhost:5432/db', description: 'Database connection' },
  { key: 'API_KEY', value: '', description: 'API key for external services' },
  { key: 'GOOS', value: 'linux', description: 'Go OS target' },
  { key: 'GOARCH', value: 'amd64', description: 'Go architecture target' },
  { key: 'PYTHONPATH', value: '/app', description: 'Python module search path' },
  { key: 'JAVA_OPTS', value: '-Xmx512m', description: 'Java runtime options' }
];

export function PipelineEnvironmentForm({ env, onChange }: PipelineEnvironmentFormProps) {
  const [envVars, setEnvVars] = useState<EnvVar[]>(() => {
    return Object.entries(env).map(([key, value]) => ({
      key,
      value,
      isSecret: key.toLowerCase().includes('key') || key.toLowerCase().includes('secret') || key.toLowerCase().includes('password')
    }));
  });
  const [visibleSecrets, setVisibleSecrets] = useState<Set<number>>(new Set());

  const updateEnvVars = (newEnvVars: EnvVar[]) => {
    setEnvVars(newEnvVars);
    const envObject = newEnvVars.reduce((acc, envVar) => {
      if (envVar.key.trim()) {
        acc[envVar.key] = envVar.value;
      }
      return acc;
    }, {} as Record<string, string>);
    onChange(envObject);
  };

  const addEnvVar = (template?: { key: string; value: string }) => {
    const newEnvVar: EnvVar = {
      key: template?.key || '',
      value: template?.value || '',
      isSecret: template?.key ? (template.key.toLowerCase().includes('key') || template.key.toLowerCase().includes('secret')) : false
    };
    updateEnvVars([...envVars, newEnvVar]);
  };

  const removeEnvVar = (index: number) => {
    updateEnvVars(envVars.filter((_, i) => i !== index));
  };

  const updateEnvVar = (index: number, field: keyof EnvVar, value: string | boolean) => {
    const newEnvVars = [...envVars];
    newEnvVars[index] = { ...newEnvVars[index], [field]: value };
    updateEnvVars(newEnvVars);
  };

  const toggleSecretVisibility = (index: number) => {
    const newVisible = new Set(visibleSecrets);
    if (newVisible.has(index)) {
      newVisible.delete(index);
    } else {
      newVisible.add(index);
    }
    setVisibleSecrets(newVisible);
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Environment Variables</h3>
            <p className="text-sm text-muted-foreground">
              Configure environment variables for your pipeline execution
            </p>
          </div>
          <Button onClick={() => addEnvVar()} className="bg-gradient-to-r from-primary to-primary-hover">
            <Plus className="h-4 w-4 mr-2" />
            Add Variable
          </Button>
        </div>
      </div>

      {/* Common Environment Variables */}
      <Card className="p-4">
        <h4 className="font-medium mb-3">Quick Add Common Variables</h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
          {commonEnvVars.map((envVar) => (
            <Button
              key={envVar.key}
              variant="outline"
              size="sm"
              onClick={() => addEnvVar(envVar)}
              className="justify-start h-auto p-2"
            >
              <div className="text-left">
                <div className="font-mono text-xs">{envVar.key}</div>
                <div className="text-xs text-muted-foreground truncate">{envVar.description}</div>
              </div>
            </Button>
          ))}
        </div>
      </Card>

      {/* Environment Variables List */}
      <div className="space-y-3">
        {envVars.map((envVar, index) => (
          <Card key={index} className="p-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="outline">#{index + 1}</Badge>
                  {envVar.isSecret && (
                    <Badge variant="destructive" className="text-xs">
                      Secret
                    </Badge>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id={`secret-${index}`}
                      checked={envVar.isSecret}
                      onCheckedChange={(checked) => updateEnvVar(index, 'isSecret', checked)}
                    />
                    <Label htmlFor={`secret-${index}`} className="text-xs">Secret</Label>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeEnvVar(index)}
                    className="text-destructive hover:text-destructive"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Variable Name</Label>
                  <Input
                    placeholder="e.g., API_KEY"
                    value={envVar.key}
                    onChange={(e) => updateEnvVar(index, 'key', e.target.value)}
                    className="font-mono"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Value</Label>
                  <div className="relative">
                    <Input
                      type={envVar.isSecret && !visibleSecrets.has(index) ? 'password' : 'text'}
                      placeholder={envVar.isSecret ? '••••••••' : 'Enter value'}
                      value={envVar.value}
                      onChange={(e) => updateEnvVar(index, 'value', e.target.value)}
                      className="pr-10"
                    />
                    {envVar.isSecret && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleSecretVisibility(index)}
                        className="absolute right-1 top-1 h-6 w-6 p-0"
                      >
                        {visibleSecrets.has(index) ? (
                          <EyeOff className="h-3 w-3" />
                        ) : (
                          <Eye className="h-3 w-3" />
                        )}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </Card>
        ))}

        {envVars.length === 0 && (
          <Card className="p-8 text-center bg-muted/30">
            <h4 className="font-medium mb-2">No environment variables</h4>
            <p className="text-sm text-muted-foreground mb-4">
              Add environment variables to configure your pipeline execution
            </p>
            <Button onClick={() => addEnvVar()} variant="outline">
              <Plus className="h-4 w-4 mr-2" />
              Add First Variable
            </Button>
          </Card>
        )}
      </div>
    </div>
  );
}