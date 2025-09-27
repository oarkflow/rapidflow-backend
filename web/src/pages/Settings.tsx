import { useState } from 'react';
import { Save, RefreshCw, Bell, Shield, Database, Palette } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/hooks/use-toast';

export default function Settings() {
  const { toast } = useToast();
  
  const [settings, setSettings] = useState({
    general: {
      platformName: 'DevOps CI/CD Platform',
      defaultTimeout: '30',
      maxConcurrentJobs: '5',
      retentionDays: '30',
      timezone: 'UTC'
    },
    notifications: {
      emailEnabled: true,
      slackEnabled: false,
      webhookEnabled: false,
      emailHost: 'smtp.gmail.com',
      emailPort: '587',
      emailUser: '',
      slackWebhook: '',
      webhookUrl: ''
    },
    security: {
      requireMFA: false,
      sessionTimeout: '24',
      maxLoginAttempts: '5',
      allowGuestAccess: false,
      apiKeyExpiry: '365'
    },
    database: {
      host: 'localhost',
      port: '5432',
      database: 'cicd_platform',
      connectionPool: '10',
      backupEnabled: true,
      backupFrequency: 'daily'
    },
    ui: {
      theme: 'light',
      compactMode: false,
      showJobDetails: true,
      autoRefresh: true,
      refreshInterval: '5'
    }
  });

  const updateSetting = (category: keyof typeof settings, key: string, value: any) => {
    setSettings(prev => ({
      ...prev,
      [category]: {
        ...prev[category],
        [key]: value
      }
    }));
  };

  const saveSettings = () => {
    // Here you would typically save to backend
    toast({
      title: "Settings Saved",
      description: "Your platform settings have been updated successfully.",
    });
  };

  const resetSettings = () => {
    // Reset to defaults
    toast({
      title: "Settings Reset",
      description: "Settings have been reset to default values.",
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-foreground">Platform Settings</h1>
          <p className="text-muted-foreground mt-1">
            Configure your CI/CD platform preferences and behavior
          </p>
        </div>
        
        <div className="flex gap-2">
          <Button variant="outline" onClick={resetSettings}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Reset to Defaults
          </Button>
          <Button onClick={saveSettings} className="bg-gradient-to-r from-primary to-primary-hover">
            <Save className="h-4 w-4 mr-2" />
            Save Settings
          </Button>
        </div>
      </div>

      <Tabs defaultValue="general" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="database">Database</TabsTrigger>
          <TabsTrigger value="ui">Interface</TabsTrigger>
        </TabsList>

        <TabsContent value="general">
          <Card className="p-6">
            <div className="space-y-6">
              <div className="flex items-center gap-2 mb-4">
                <Shield className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">General Settings</h3>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label>Platform Name</Label>
                  <Input
                    value={settings.general.platformName}
                    onChange={(e) => updateSetting('general', 'platformName', e.target.value)}
                    placeholder="DevOps CI/CD Platform"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Default Job Timeout (minutes)</Label>
                  <Input
                    type="number"
                    value={settings.general.defaultTimeout}
                    onChange={(e) => updateSetting('general', 'defaultTimeout', e.target.value)}
                    placeholder="30"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Max Concurrent Jobs</Label>
                  <Input
                    type="number"
                    value={settings.general.maxConcurrentJobs}
                    onChange={(e) => updateSetting('general', 'maxConcurrentJobs', e.target.value)}
                    placeholder="5"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Log Retention (days)</Label>
                  <Input
                    type="number"
                    value={settings.general.retentionDays}
                    onChange={(e) => updateSetting('general', 'retentionDays', e.target.value)}
                    placeholder="30"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label>Timezone</Label>
                  <Select
                    value={settings.general.timezone}
                    onValueChange={(value) => updateSetting('general', 'timezone', value)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="UTC">UTC</SelectItem>
                      <SelectItem value="America/New_York">Eastern Time</SelectItem>
                      <SelectItem value="America/Chicago">Central Time</SelectItem>
                      <SelectItem value="America/Los_Angeles">Pacific Time</SelectItem>
                      <SelectItem value="Europe/London">London</SelectItem>
                      <SelectItem value="Europe/Paris">Paris</SelectItem>
                      <SelectItem value="Asia/Tokyo">Tokyo</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="notifications">
          <Card className="p-6">
            <div className="space-y-6">
              <div className="flex items-center gap-2 mb-4">
                <Bell className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Notification Settings</h3>
              </div>
              
              <div className="space-y-6">
                {/* Email Notifications */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Email Notifications</h4>
                      <p className="text-sm text-muted-foreground">Send job status updates via email</p>
                    </div>
                    <Switch
                      checked={settings.notifications.emailEnabled}
                      onCheckedChange={(checked) => updateSetting('notifications', 'emailEnabled', checked)}
                    />
                  </div>
                  
                  {settings.notifications.emailEnabled && (
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 ml-4 border-l-2 border-muted pl-4">
                      <div className="space-y-2">
                        <Label>SMTP Host</Label>
                        <Input
                          value={settings.notifications.emailHost}
                          onChange={(e) => updateSetting('notifications', 'emailHost', e.target.value)}
                          placeholder="smtp.gmail.com"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>SMTP Port</Label>
                        <Input
                          value={settings.notifications.emailPort}
                          onChange={(e) => updateSetting('notifications', 'emailPort', e.target.value)}
                          placeholder="587"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>SMTP Username</Label>
                        <Input
                          value={settings.notifications.emailUser}
                          onChange={(e) => updateSetting('notifications', 'emailUser', e.target.value)}
                          placeholder="your@email.com"
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* Slack Notifications */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Slack Notifications</h4>
                      <p className="text-sm text-muted-foreground">Send notifications to Slack channels</p>
                    </div>
                    <Switch
                      checked={settings.notifications.slackEnabled}
                      onCheckedChange={(checked) => updateSetting('notifications', 'slackEnabled', checked)}
                    />
                  </div>
                  
                  {settings.notifications.slackEnabled && (
                    <div className="ml-4 border-l-2 border-muted pl-4">
                      <div className="space-y-2">
                        <Label>Webhook URL</Label>
                        <Input
                          type="password"
                          value={settings.notifications.slackWebhook}
                          onChange={(e) => updateSetting('notifications', 'slackWebhook', e.target.value)}
                          placeholder="https://hooks.slack.com/services/..."
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* Webhook Notifications */}
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Webhook Notifications</h4>
                      <p className="text-sm text-muted-foreground">Send HTTP notifications to external services</p>
                    </div>
                    <Switch
                      checked={settings.notifications.webhookEnabled}
                      onCheckedChange={(checked) => updateSetting('notifications', 'webhookEnabled', checked)}
                    />
                  </div>
                  
                  {settings.notifications.webhookEnabled && (
                    <div className="ml-4 border-l-2 border-muted pl-4">
                      <div className="space-y-2">
                        <Label>Webhook URL</Label>
                        <Input
                          value={settings.notifications.webhookUrl}
                          onChange={(e) => updateSetting('notifications', 'webhookUrl', e.target.value)}
                          placeholder="https://api.example.com/webhook"
                        />
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="security">
          <Card className="p-6">
            <div className="space-y-6">
              <div className="flex items-center gap-2 mb-4">
                <Shield className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Security Settings</h3>
              </div>
              
              <div className="space-y-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-medium">Require Multi-Factor Authentication</h4>
                    <p className="text-sm text-muted-foreground">Enforce MFA for all user accounts</p>
                  </div>
                  <Switch
                    checked={settings.security.requireMFA}
                    onCheckedChange={(checked) => updateSetting('security', 'requireMFA', checked)}
                  />
                </div>
                
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-medium">Allow Guest Access</h4>
                    <p className="text-sm text-muted-foreground">Allow read-only access without authentication</p>
                  </div>
                  <Switch
                    checked={settings.security.allowGuestAccess}
                    onCheckedChange={(checked) => updateSetting('security', 'allowGuestAccess', checked)}
                  />
                </div>
                
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <Label>Session Timeout (hours)</Label>
                    <Input
                      type="number"
                      value={settings.security.sessionTimeout}
                      onChange={(e) => updateSetting('security', 'sessionTimeout', e.target.value)}
                      placeholder="24"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label>Max Login Attempts</Label>
                    <Input
                      type="number"
                      value={settings.security.maxLoginAttempts}
                      onChange={(e) => updateSetting('security', 'maxLoginAttempts', e.target.value)}
                      placeholder="5"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label>API Key Expiry (days)</Label>
                    <Input
                      type="number"
                      value={settings.security.apiKeyExpiry}
                      onChange={(e) => updateSetting('security', 'apiKeyExpiry', e.target.value)}
                      placeholder="365"
                    />
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="database">
          <Card className="p-6">
            <div className="space-y-6">
              <div className="flex items-center gap-2 mb-4">
                <Database className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Database Settings</h3>
              </div>
              
              <div className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Database Host</Label>
                    <Input
                      value={settings.database.host}
                      onChange={(e) => updateSetting('database', 'host', e.target.value)}
                      placeholder="localhost"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label>Database Port</Label>
                    <Input
                      value={settings.database.port}
                      onChange={(e) => updateSetting('database', 'port', e.target.value)}
                      placeholder="5432"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label>Database Name</Label>
                    <Input
                      value={settings.database.database}
                      onChange={(e) => updateSetting('database', 'database', e.target.value)}
                      placeholder="cicd_platform"
                    />
                  </div>
                  
                  <div className="space-y-2">
                    <Label>Connection Pool Size</Label>
                    <Input
                      type="number"
                      value={settings.database.connectionPool}
                      onChange={(e) => updateSetting('database', 'connectionPool', e.target.value)}
                      placeholder="10"
                    />
                  </div>
                </div>
                
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Enable Automatic Backups</h4>
                      <p className="text-sm text-muted-foreground">Automatically backup database on schedule</p>
                    </div>
                    <Switch
                      checked={settings.database.backupEnabled}
                      onCheckedChange={(checked) => updateSetting('database', 'backupEnabled', checked)}
                    />
                  </div>
                  
                  {settings.database.backupEnabled && (
                    <div className="ml-4 border-l-2 border-muted pl-4">
                      <div className="space-y-2">
                        <Label>Backup Frequency</Label>
                        <Select
                          value={settings.database.backupFrequency}
                          onValueChange={(value) => updateSetting('database', 'backupFrequency', value)}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="hourly">Hourly</SelectItem>
                            <SelectItem value="daily">Daily</SelectItem>
                            <SelectItem value="weekly">Weekly</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>

        <TabsContent value="ui">
          <Card className="p-6">
            <div className="space-y-6">
              <div className="flex items-center gap-2 mb-4">
                <Palette className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">User Interface Settings</h3>
              </div>
              
              <div className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label>Theme</Label>
                    <Select
                      value={settings.ui.theme}
                      onValueChange={(value) => updateSetting('ui', 'theme', value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="light">Light</SelectItem>
                        <SelectItem value="dark">Dark</SelectItem>
                        <SelectItem value="system">System</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  
                  <div className="space-y-2">
                    <Label>Auto-refresh Interval (seconds)</Label>
                    <Input
                      type="number"
                      value={settings.ui.refreshInterval}
                      onChange={(e) => updateSetting('ui', 'refreshInterval', e.target.value)}
                      placeholder="5"
                      disabled={!settings.ui.autoRefresh}
                    />
                  </div>
                </div>
                
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Compact Mode</h4>
                      <p className="text-sm text-muted-foreground">Use compact layout for better space utilization</p>
                    </div>
                    <Switch
                      checked={settings.ui.compactMode}
                      onCheckedChange={(checked) => updateSetting('ui', 'compactMode', checked)}
                    />
                  </div>
                  
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Show Job Details</h4>
                      <p className="text-sm text-muted-foreground">Display detailed information on job cards</p>
                    </div>
                    <Switch
                      checked={settings.ui.showJobDetails}
                      onCheckedChange={(checked) => updateSetting('ui', 'showJobDetails', checked)}
                    />
                  </div>
                  
                  <div className="flex items-center justify-between">
                    <div>
                      <h4 className="font-medium">Auto-refresh</h4>
                      <p className="text-sm text-muted-foreground">Automatically refresh job status and logs</p>
                    </div>
                    <Switch
                      checked={settings.ui.autoRefresh}
                      onCheckedChange={(checked) => updateSetting('ui', 'autoRefresh', checked)}
                    />
                  </div>
                </div>
              </div>
            </div>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}