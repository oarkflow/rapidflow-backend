import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { PipelineConfig } from '@/services/api';

interface PipelineBasicFormProps {
  config: PipelineConfig;
  onChange: (updates: Partial<PipelineConfig>) => void;
}

const languageVersions = {
  nodejs: ['16', '18', '20', '21'],
  golang: ['1.19', '1.20', '1.21', '1.22'],
  python: ['3.8', '3.9', '3.10', '3.11', '3.12'],
  java: ['11', '17', '21'],
  dotnet: ['6.0', '7.0', '8.0'],
  ruby: ['2.7', '3.0', '3.1', '3.2'],
  php: ['7.4', '8.0', '8.1', '8.2', '8.3']
};

export function PipelineBasicForm({ config, onChange }: PipelineBasicFormProps) {
  const updateField = (field: keyof PipelineConfig, value: any) => {
    onChange({ [field]: value });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Basic Configuration</h3>
        <p className="text-sm text-muted-foreground">
          Set up the fundamental properties of your pipeline
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-2">
          <Label htmlFor="name">Pipeline Name *</Label>
          <Input
            id="name"
            placeholder="e.g., Frontend Build Pipeline"
            value={config.name}
            onChange={(e) => updateField('name', e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="language">Language</Label>
          <Select
            value={config.language || ''}
            onValueChange={(value) => {
              updateField('language', value);
              // Reset version when language changes
              const versions = languageVersions[value as keyof typeof languageVersions];
              if (versions) {
                updateField('version', versions[versions.length - 1]);
              }
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select language" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="nodejs">Node.js</SelectItem>
              <SelectItem value="golang">Go</SelectItem>
              <SelectItem value="python">Python</SelectItem>
              <SelectItem value="java">Java</SelectItem>
              <SelectItem value="dotnet">.NET</SelectItem>
              <SelectItem value="ruby">Ruby</SelectItem>
              <SelectItem value="php">PHP</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="version">Version</Label>
          <Select
            value={config.version || ''}
            onValueChange={(value) => updateField('version', value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select version" />
            </SelectTrigger>
            <SelectContent>
              {config.language && languageVersions[config.language as keyof typeof languageVersions]?.map((version) => (
                <SelectItem key={version} value={version}>
                  {version}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="branch">Branch</Label>
          <Input
            id="branch"
            placeholder="main"
            value={config.branch || ''}
            onChange={(e) => updateField('branch', e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="folder">Working Directory</Label>
          <Input
            id="folder"
            placeholder="./"
            value={config.folder || ''}
            onChange={(e) => updateField('folder', e.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="repo_url">Repository URL</Label>
          <Input
            id="repo_url"
            placeholder="https://github.com/user/repo.git"
            value={config.repo_url || ''}
            onChange={(e) => updateField('repo_url', e.target.value)}
          />
        </div>
      </div>

      <div className="space-y-4">
        <h4 className="font-medium">Options</h4>
        
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <Label htmlFor="expose_ports">Expose Ports</Label>
            <p className="text-sm text-muted-foreground">
              Allow external access to container ports
            </p>
          </div>
          <Switch
            id="expose_ports"
            checked={config.expose_ports || false}
            onCheckedChange={(checked) => updateField('expose_ports', checked)}
          />
        </div>

        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <Label htmlFor="temporary">Temporary Build</Label>
            <p className="text-sm text-muted-foreground">
              Clean up resources immediately after completion
            </p>
          </div>
          <Switch
            id="temporary"
            checked={config.temporary || false}
            onCheckedChange={(checked) => updateField('temporary', checked)}
          />
        </div>
      </div>
    </div>
  );
}