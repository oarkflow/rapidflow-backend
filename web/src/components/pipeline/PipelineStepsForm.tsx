import { useState } from 'react';
import { Plus, Trash2, GripVertical, Play } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';

interface Step {
  type: string;
  content: string;
  files?: Record<string, string>;
}

interface PipelineStepsFormProps {
  steps: Step[];
  onChange: (steps: Step[]) => void;
}

const stepTypes = [
  { value: 'bash', label: 'Bash Script', description: 'Run shell commands' },
  { value: 'docker', label: 'Docker Command', description: 'Execute Docker operations' },
  { value: 'git', label: 'Git Command', description: 'Version control operations' },
  { value: 'file', label: 'File Operation', description: 'Create or modify files' },
  { value: 'test', label: 'Test Runner', description: 'Execute test suites' },
  { value: 'build', label: 'Build Command', description: 'Compile or package code' }
];

const stepTemplates = {
  bash: 'echo "Hello World"\nls -la\npwd',
  docker: 'docker build -t myapp .\ndocker run --rm myapp',
  git: 'git clone $REPO_URL\ncd repository\ngit checkout $BRANCH',
  file: 'mkdir -p dist\ncp src/* dist/',
  test: 'npm test\n# or\npython -m pytest\n# or\ngo test ./...',
  build: 'npm run build\n# or\ngo build -o app .\n# or\nmvn package'
};

export function PipelineStepsForm({ steps, onChange }: PipelineStepsFormProps) {
  const [expandedStep, setExpandedStep] = useState<number | null>(0);

  const addStep = () => {
    const newStep: Step = {
      type: 'bash',
      content: stepTemplates.bash,
      files: {}
    };
    onChange([...steps, newStep]);
    setExpandedStep(steps.length);
  };

  const removeStep = (index: number) => {
    const newSteps = steps.filter((_, i) => i !== index);
    onChange(newSteps);
    if (expandedStep === index) {
      setExpandedStep(null);
    }
  };

  const updateStep = (index: number, field: keyof Step, value: any) => {
    const newSteps = [...steps];
    newSteps[index] = { ...newSteps[index], [field]: value };
    onChange(newSteps);
  };

  const moveStep = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= steps.length) return;
    
    const newSteps = [...steps];
    [newSteps[index], newSteps[newIndex]] = [newSteps[newIndex], newSteps[index]];
    onChange(newSteps);
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Pipeline Steps</h3>
            <p className="text-sm text-muted-foreground">
              Define the sequence of operations for your pipeline
            </p>
          </div>
          <Button onClick={addStep} className="bg-gradient-to-r from-primary to-primary-hover">
            <Plus className="h-4 w-4 mr-2" />
            Add Step
          </Button>
        </div>
      </div>

      {steps.length === 0 ? (
        <Card className="p-8 text-center bg-muted/30">
          <Play className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
          <h4 className="font-medium mb-2">No steps defined</h4>
          <p className="text-sm text-muted-foreground mb-4">
            Add your first step to start building your pipeline
          </p>
          <Button onClick={addStep} variant="outline">
            <Plus className="h-4 w-4 mr-2" />
            Add First Step
          </Button>
        </Card>
      ) : (
        <div className="space-y-4">
          {steps.map((step, index) => (
            <Card key={index} className="p-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex flex-col gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0"
                        onClick={() => moveStep(index, 'up')}
                        disabled={index === 0}
                      >
                        ▲
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0"
                        onClick={() => moveStep(index, 'down')}
                        disabled={index === steps.length - 1}
                      >
                        ▼
                      </Button>
                    </div>
                    <GripVertical className="h-4 w-4 text-muted-foreground" />
                    <div className="flex items-center gap-2">
                      <Badge variant="outline">Step {index + 1}</Badge>
                      <Badge variant="secondary">
                        {stepTypes.find(t => t.value === step.type)?.label || step.type}
                      </Badge>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setExpandedStep(expandedStep === index ? null : index)}
                    >
                      {expandedStep === index ? 'Collapse' : 'Expand'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeStep(index)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {expandedStep === index && (
                  <div className="space-y-4 border-t pt-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>Step Type</Label>
                        <Select
                          value={step.type}
                          onValueChange={(value) => {
                            updateStep(index, 'type', value);
                            updateStep(index, 'content', stepTemplates[value as keyof typeof stepTemplates] || '');
                          }}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {stepTypes.map((type) => (
                              <SelectItem key={type.value} value={type.value}>
                                <div>
                                  <div className="font-medium">{type.label}</div>
                                  <div className="text-xs text-muted-foreground">{type.description}</div>
                                </div>
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    <div className="space-y-2">
                      <Label>Script Content</Label>
                      <Textarea
                        placeholder="Enter your script content..."
                        value={step.content}
                        onChange={(e) => updateStep(index, 'content', e.target.value)}
                        className="font-mono text-sm"
                        rows={6}
                      />
                    </div>
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