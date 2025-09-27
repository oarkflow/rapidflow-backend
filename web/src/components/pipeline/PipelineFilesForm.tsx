import { useState } from 'react';
import { Plus, Trash2, FileText, Edit } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';

interface Step {
  type: string;
  content: string;
  files?: Record<string, string>;
}

interface PipelineFilesFormProps {
  steps: Step[];
  onChange: (steps: Step[]) => void;
}

const fileTemplates = {
  dockerfile: `FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 3000
CMD ["npm", "start"]`,
  
  'docker-compose.yml': `version: '3.8'
services:
  app:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production`,
      
  'nginx.conf': `server {
    listen 80;
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}`,

  'config.json': `{
  "port": 3000,
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "myapp"
  },
  "features": {
    "enableCache": true,
    "enableLogs": true
  }
}`,

  '.env.example': `NODE_ENV=production
PORT=3000
DATABASE_URL=postgresql://user:pass@localhost:5432/db
API_KEY=your_api_key_here
SECRET_KEY=your_secret_key_here`
};

export function PipelineFilesForm({ steps, onChange }: PipelineFilesFormProps) {
  const [selectedStep, setSelectedStep] = useState(0);
  const [editingFile, setEditingFile] = useState<{ stepIndex: number; fileName: string; content: string } | null>(null);
  const [newFileName, setNewFileName] = useState('');

  const updateStepFiles = (stepIndex: number, files: Record<string, string>) => {
    const newSteps = [...steps];
    newSteps[stepIndex] = { ...newSteps[stepIndex], files };
    onChange(newSteps);
  };

  const addFile = (stepIndex: number, fileName: string, content: string = '') => {
    const step = steps[stepIndex];
    const newFiles = { ...step.files, [fileName]: content };
    updateStepFiles(stepIndex, newFiles);
    setNewFileName('');
  };

  const removeFile = (stepIndex: number, fileName: string) => {
    const step = steps[stepIndex];
    const newFiles = { ...step.files };
    delete newFiles[fileName];
    updateStepFiles(stepIndex, newFiles);
  };

  const updateFile = (stepIndex: number, fileName: string, content: string) => {
    const step = steps[stepIndex];
    const newFiles = { ...step.files, [fileName]: content };
    updateStepFiles(stepIndex, newFiles);
  };

  const saveEditingFile = () => {
    if (editingFile) {
      updateFile(editingFile.stepIndex, editingFile.fileName, editingFile.content);
      setEditingFile(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Pipeline Files</h3>
        <p className="text-sm text-muted-foreground">
          Add configuration files, scripts, and other resources needed for your pipeline steps
        </p>
      </div>

      {/* Step Selector */}
      <Card className="p-4">
        <h4 className="font-medium mb-3">Select Step to Add Files</h4>
        <div className="flex flex-wrap gap-2">
          {steps.map((step, index) => (
            <Button
              key={index}
              variant={selectedStep === index ? "default" : "outline"}
              size="sm"
              onClick={() => setSelectedStep(index)}
            >
              Step {index + 1}: {step.type}
            </Button>
          ))}
        </div>
      </Card>

      {steps.length > 0 && (
        <Card className="p-4">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="font-medium">
                Files for Step {selectedStep + 1} ({steps[selectedStep].type})
              </h4>
              <div className="flex items-center gap-2">
                <Input
                  placeholder="Enter filename..."
                  value={newFileName}
                  onChange={(e) => setNewFileName(e.target.value)}
                  className="w-48"
                />
                <Button
                  onClick={() => addFile(selectedStep, newFileName)}
                  disabled={!newFileName.trim()}
                  size="sm"
                >
                  <Plus className="h-4 w-4 mr-1" />
                  Add File
                </Button>
              </div>
            </div>

            {/* File Templates */}
            <div className="space-y-2">
              <Label className="text-sm">Quick Add Templates</Label>
              <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
                {Object.entries(fileTemplates).map(([fileName, content]) => (
                  <Button
                    key={fileName}
                    variant="outline"
                    size="sm"
                    onClick={() => addFile(selectedStep, fileName, content)}
                    className="justify-start"
                  >
                    <FileText className="h-3 w-3 mr-1" />
                    {fileName}
                  </Button>
                ))}
              </div>
            </div>

            {/* Files List */}
            <div className="space-y-2">
              {Object.entries(steps[selectedStep].files || {}).map(([fileName, content]) => (
                <div key={fileName} className="flex items-center justify-between p-3 border rounded-lg">
                  <div className="flex items-center gap-3">
                    <FileText className="h-4 w-4 text-muted-foreground" />
                    <span className="font-mono text-sm">{fileName}</span>
                    <Badge variant="secondary" className="text-xs">
                      {content.split('\n').length} lines
                    </Badge>
                  </div>
                  <div className="flex items-center gap-1">
                    <Dialog>
                      <DialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setEditingFile({ stepIndex: selectedStep, fileName, content })}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-w-4xl max-h-[80vh]">
                        <DialogHeader>
                          <DialogTitle>Edit {fileName}</DialogTitle>
                        </DialogHeader>
                        <div className="space-y-4">
                          <Textarea
                            value={editingFile?.fileName === fileName ? editingFile.content : content}
                            onChange={(e) => setEditingFile(prev => 
                              prev ? { ...prev, content: e.target.value } : null
                            )}
                            className="font-mono text-sm min-h-[400px]"
                            placeholder="Enter file content..."
                          />
                          <div className="flex justify-end gap-2">
                            <Button variant="outline" onClick={() => setEditingFile(null)}>
                              Cancel
                            </Button>
                            <Button onClick={saveEditingFile}>
                              Save Changes
                            </Button>
                          </div>
                        </div>
                      </DialogContent>
                    </Dialog>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeFile(selectedStep, fileName)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}

              {(!steps[selectedStep].files || Object.keys(steps[selectedStep].files).length === 0) && (
                <div className="text-center py-8 text-muted-foreground">
                  <FileText className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No files added to this step yet</p>
                </div>
              )}
            </div>
          </div>
        </Card>
      )}

      {steps.length === 0 && (
        <Card className="p-8 text-center bg-muted/30">
          <FileText className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
          <h4 className="font-medium mb-2">No steps available</h4>
          <p className="text-sm text-muted-foreground">
            Add pipeline steps first to configure files
          </p>
        </Card>
      )}
    </div>
  );
}