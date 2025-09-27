import { useState } from 'react';
import { Github, GitBranch, Cloud, Database, Settings2, Key } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { PipelineConfig } from '@/services/api';

interface PipelineProvidersFormProps {
  config: PipelineConfig;
  onChange: (updates: Partial<PipelineConfig>) => void;
}

interface ProviderConfig {
  git?: {
    provider: string;
    repository: string;
    branch: string;
    credentials?: {
      username: string;
      token: string;
    };
  };
  cloud?: {
    provider: string;
    region: string;
    credentials?: {
      accessKey: string;
      secretKey: string;
    };
  };
  database?: {
    provider: string;
    host: string;
    port: string;
    database: string;
    credentials?: {
      username: string;
      password: string;
    };
  };
  notifications?: {
    slack?: {
      webhook: string;
      channel: string;
    };
    email?: {
      smtp: string;
      recipients: string[];
    };
  };
}

const gitProviders = [
  { value: 'github', label: 'GitHub', icon: Github },
  { value: 'gitlab', label: 'GitLab', icon: GitBranch },
  { value: 'bitbucket', label: 'Bitbucket', icon: GitBranch },
  { value: 'custom', label: 'Custom Git', icon: GitBranch }
];

const cloudProviders = [
  { value: 'aws', label: 'Amazon AWS' },
  { value: 'gcp', label: 'Google Cloud Platform' },
  { value: 'azure', label: 'Microsoft Azure' },
  { value: 'digitalocean', label: 'DigitalOcean' },
  { value: 'custom', label: 'Custom Cloud' }
];

const databaseProviders = [
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'mongodb', label: 'MongoDB' },
  { value: 'redis', label: 'Redis' },
  { value: 'sqlite', label: 'SQLite' }
];

export function PipelineProvidersForm({ config, onChange }: PipelineProvidersFormProps) {
  const [providers, setProviders] = useState<ProviderConfig>({
    git: {
      provider: 'github',
      repository: config.repo_url || '',
      branch: config.branch || 'main'
    }
  });

  const [enabledProviders, setEnabledProviders] = useState({
    git: !!config.repo_url,
    cloud: false,
    database: false,
    notifications: false
  });

  const updateProvider = (providerType: keyof ProviderConfig, updates: any) => {
    const newProviders = {
      ...providers,
      [providerType]: { ...providers[providerType], ...updates }
    };
    setProviders(newProviders);

    // Update main config if needed
    if (providerType === 'git') {
      onChange({
        repo_url: newProviders.git?.repository,
        branch: newProviders.git?.branch
      });
    }
  };

  const toggleProvider = (providerType: keyof typeof enabledProviders) => {
    setEnabledProviders(prev => ({
      ...prev,
      [providerType]: !prev[providerType]
    }));
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">External Providers & Integrations</h3>
        <p className="text-sm text-muted-foreground">
          Configure external services and providers for your pipeline
        </p>
      </div>

      <Tabs defaultValue="git" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="git">Git Repositories</TabsTrigger>
          <TabsTrigger value="cloud">Cloud Providers</TabsTrigger>
          <TabsTrigger value="database">Databases</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
        </TabsList>

        <TabsContent value="git">
          <Card className="p-6">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Github className="h-5 w-5" />
                  <h4 className="font-medium">Git Repository Configuration</h4>
                </div>
                <Switch
                  checked={enabledProviders.git}
                  onCheckedChange={() => toggleProvider('git')}
                />
              </div>

              {enabledProviders.git && (
                <div className="space-y-4 border-t pt-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Git Provider</Label>
                      <Select
                        value={providers.git?.provider}
                        onValueChange={(value) => updateProvider('git', { provider: value })}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {gitProviders.map((provider) => (
                            <SelectItem key={provider.value} value={provider.value}>
                              <div className="flex items-center gap-2">
                                <provider.icon className="h-4 w-4" />
                                {provider.label}
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label>Branch</Label>
                      <Input
                        value={providers.git?.branch || ''}
                        onChange={(e) => updateProvider('git', { branch: e.target.value })}
                        placeholder="main"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>Repository URL</Label>
                    <Input
                      value={providers.git?.repository || ''}
                      onChange={(e) => updateProvider('git', { repository: e.target.value })}
                      placeholder="https://github.com/user/repo.git"
                    />
                  </div>

                  <div className="space-y-4">
                    <h5 className="font-medium text-sm">Authentication (Optional)</h5>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>Username</Label>
                        <Input
                          value={providers.git?.credentials?.username || ''}
                          onChange={(e) => updateProvider('git', { 
                            credentials: { 
                              ...providers.git?.credentials, 
                              username: e.target.value 
                            }
                          })}
                          placeholder="your-username"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Access Token</Label>
                        <Input
                          type="password"
                          value={providers.git?.credentials?.token || ''}
                          onChange={(e) => updateProvider('git', { 
                            credentials: { 
                              ...providers.git?.credentials, 
                              token: e.target.value 
                            }
                          })}
                          placeholder="••••••••••••••••"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="cloud">
          <Card className="p-6">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Cloud className="h-5 w-5" />
                  <h4 className="font-medium">Cloud Provider Configuration</h4>
                </div>
                <Switch
                  checked={enabledProviders.cloud}
                  onCheckedChange={() => toggleProvider('cloud')}
                />
              </div>

              {enabledProviders.cloud && (
                <div className="space-y-4 border-t pt-4">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>Cloud Provider</Label>
                      <Select
                        value={providers.cloud?.provider}
                        onValueChange={(value) => updateProvider('cloud', { provider: value })}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select provider" />
                        </SelectTrigger>
                        <SelectContent>
                          {cloudProviders.map((provider) => (
                            <SelectItem key={provider.value} value={provider.value}>
                              {provider.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label>Region</Label>
                      <Input
                        value={providers.cloud?.region || ''}
                        onChange={(e) => updateProvider('cloud', { region: e.target.value })}
                        placeholder="us-east-1"
                      />
                    </div>
                  </div>

                  <div className="space-y-4">
                    <h5 className="font-medium text-sm">Credentials</h5>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>Access Key</Label>
                        <Input
                          value={providers.cloud?.credentials?.accessKey || ''}
                          onChange={(e) => updateProvider('cloud', { 
                            credentials: { 
                              ...providers.cloud?.credentials, 
                              accessKey: e.target.value 
                            }
                          })}
                          placeholder="AKIA..."
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Secret Key</Label>
                        <Input
                          type="password"
                          value={providers.cloud?.credentials?.secretKey || ''}
                          onChange={(e) => updateProvider('cloud', { 
                            credentials: { 
                              ...providers.cloud?.credentials, 
                              secretKey: e.target.value 
                            }
                          })}
                          placeholder="••••••••••••••••"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="database">
          <Card className="p-6">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Database className="h-5 w-5" />
                  <h4 className="font-medium">Database Configuration</h4>
                </div>
                <Switch
                  checked={enabledProviders.database}
                  onCheckedChange={() => toggleProvider('database')}
                />
              </div>

              {enabledProviders.database && (
                <div className="space-y-4 border-t pt-4">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="space-y-2">
                      <Label>Database Type</Label>
                      <Select
                        value={providers.database?.provider}
                        onValueChange={(value) => updateProvider('database', { provider: value })}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select database" />
                        </SelectTrigger>
                        <SelectContent>
                          {databaseProviders.map((provider) => (
                            <SelectItem key={provider.value} value={provider.value}>
                              {provider.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label>Host</Label>
                      <Input
                        value={providers.database?.host || ''}
                        onChange={(e) => updateProvider('database', { host: e.target.value })}
                        placeholder="localhost"
                      />
                    </div>

                    <div className="space-y-2">
                      <Label>Port</Label>
                      <Input
                        value={providers.database?.port || ''}
                        onChange={(e) => updateProvider('database', { port: e.target.value })}
                        placeholder="5432"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>Database Name</Label>
                    <Input
                      value={providers.database?.database || ''}
                      onChange={(e) => updateProvider('database', { database: e.target.value })}
                      placeholder="myapp"
                    />
                  </div>

                  <div className="space-y-4">
                    <h5 className="font-medium text-sm">Authentication</h5>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>Username</Label>
                        <Input
                          value={providers.database?.credentials?.username || ''}
                          onChange={(e) => updateProvider('database', { 
                            credentials: { 
                              ...providers.database?.credentials, 
                              username: e.target.value 
                            }
                          })}
                          placeholder="postgres"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Password</Label>
                        <Input
                          type="password"
                          value={providers.database?.credentials?.password || ''}
                          onChange={(e) => updateProvider('database', { 
                            credentials: { 
                              ...providers.database?.credentials, 
                              password: e.target.value 
                            }
                          })}
                          placeholder="••••••••"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="notifications">
          <Card className="p-6">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Settings2 className="h-5 w-5" />
                  <h4 className="font-medium">Notification Settings</h4>
                </div>
                <Switch
                  checked={enabledProviders.notifications}
                  onCheckedChange={() => toggleProvider('notifications')}
                />
              </div>

              {enabledProviders.notifications && (
                <div className="space-y-6 border-t pt-4">
                  {/* Slack Notifications */}
                  <div className="space-y-4">
                    <h5 className="font-medium text-sm flex items-center gap-2">
                      <Badge variant="outline">Slack</Badge>
                    </h5>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>Webhook URL</Label>
                        <Input
                          value={providers.notifications?.slack?.webhook || ''}
                          onChange={(e) => updateProvider('notifications', { 
                            slack: { 
                              ...providers.notifications?.slack, 
                              webhook: e.target.value 
                            }
                          })}
                          placeholder="https://hooks.slack.com/..."
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Channel</Label>
                        <Input
                          value={providers.notifications?.slack?.channel || ''}
                          onChange={(e) => updateProvider('notifications', { 
                            slack: { 
                              ...providers.notifications?.slack, 
                              channel: e.target.value 
                            }
                          })}
                          placeholder="#deployments"
                        />
                      </div>
                    </div>
                  </div>

                  {/* Email Notifications */}
                  <div className="space-y-4">
                    <h5 className="font-medium text-sm flex items-center gap-2">
                      <Badge variant="outline">Email</Badge>
                    </h5>
                    <div className="space-y-2">
                      <Label>SMTP Server</Label>
                      <Input
                        value={providers.notifications?.email?.smtp || ''}
                        onChange={(e) => updateProvider('notifications', { 
                          email: { 
                            ...providers.notifications?.email, 
                            smtp: e.target.value 
                          }
                        })}
                        placeholder="smtp.gmail.com:587"
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}