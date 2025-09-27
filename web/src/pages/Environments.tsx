import React, { useState } from 'react';
import { Plus, Trash2, Server, Database, Globe, Settings2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { useToast } from '@/hooks/use-toast';

interface Environment {
    id: string;
    name: string;
    type: 'development' | 'staging' | 'production';
    description: string;
    variables: Record<string, string>;
    endpoints: {
        api?: string;
        database?: string;
        cache?: string;
    };
}

const environmentTypes = [
    { value: 'development', label: 'Development', icon: Settings2, color: 'bg-blue-500' },
    { value: 'staging', label: 'Staging', icon: Globe, color: 'bg-green-500' },
    { value: 'production', label: 'Production', icon: Server, color: 'bg-red-500' }
];

export default function Environments() {
    const [environments, setEnvironments] = useState<Environment[]>([
        {
            id: '1',
            name: 'Development',
            type: 'development',
            description: 'Local development environment',
            variables: {
                NODE_ENV: 'development',
                API_URL: 'http://localhost:3000',
                DATABASE_URL: 'postgresql://dev:dev@localhost:5432/myapp_dev'
            },
            endpoints: {
                api: 'http://localhost:3000',
                database: 'localhost:5432',
                cache: 'localhost:6379'
            }
        },
        {
            id: '2',
            name: 'Production',
            type: 'production',
            description: 'Live production environment',
            variables: {
                NODE_ENV: 'production',
                API_URL: 'https://api.myapp.com',
                DATABASE_URL: 'postgresql://prod:***@prod-db:5432/myapp'
            },
            endpoints: {
                api: 'https://api.myapp.com',
                database: 'prod-db:5432',
                cache: 'prod-cache:6379'
            }
        }
    ]);

    const [selectedEnv, setSelectedEnv] = useState<Environment | null>(null);
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
    const [newEnvName, setNewEnvName] = useState('');
    const [newEnvType, setNewEnvType] = useState<'development' | 'staging' | 'production'>('development');
    const [newEnvDescription, setNewEnvDescription] = useState('');
    const { toast } = useToast();

    const createEnvironment = () => {
        if (!newEnvName.trim()) {
            toast({
                title: "Validation Error",
                description: "Environment name is required",
                variant: "destructive"
            });
            return;
        }

        const newEnv: Environment = {
            id: Date.now().toString(),
            name: newEnvName,
            type: newEnvType,
            description: newEnvDescription,
            variables: {},
            endpoints: {}
        };

        setEnvironments([...environments, newEnv]);
        setNewEnvName('');
        setNewEnvType('development');
        setNewEnvDescription('');
        setIsCreateDialogOpen(false);

        toast({
            title: "Success",
            description: `Environment "${newEnv.name}" created successfully`,
        });
    };

    const deleteEnvironment = (id: string) => {
        setEnvironments(environments.filter(env => env.id !== id));
        if (selectedEnv?.id === id) {
            setSelectedEnv(null);
        }
        toast({
            title: "Success",
            description: "Environment deleted successfully",
        });
    };

    const updateEnvironmentVariable = (envId: string, key: string, value: string) => {
        setEnvironments(environments.map(env =>
            env.id === envId
                ? { ...env, variables: { ...env.variables, [key]: value } }
                : env
        ));
    };

    const deleteEnvironmentVariable = (envId: string, key: string) => {
        setEnvironments(environments.map(env => {
            if (env.id === envId) {
                const newVariables = { ...env.variables };
                delete newVariables[key];
                return { ...env, variables: newVariables };
            }
            return env;
        }));
    };

    const getTypeConfig = (type: string) => {
        return environmentTypes.find(t => t.value === type) || environmentTypes[0];
    };

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-foreground">Environments</h1>
                    <p className="text-muted-foreground mt-1">
                        Manage deployment environments and their configurations
                    </p>
                </div>

                <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
                    <DialogTrigger asChild>
                        <Button className="bg-gradient-to-r from-primary to-primary-hover">
                            <Plus className="h-4 w-4 mr-2" />
                            Create Environment
                        </Button>
                    </DialogTrigger>
                    <DialogContent>
                        <DialogHeader>
                            <DialogTitle>Create New Environment</DialogTitle>
                        </DialogHeader>
                        <div className="space-y-4">
                            <div className="space-y-2">
                                <Label>Environment Name</Label>
                                <Input
                                    placeholder="e.g., Staging, QA, Demo"
                                    value={newEnvName}
                                    onChange={(e) => setNewEnvName(e.target.value)}
                                />
                            </div>

                            <div className="space-y-2">
                                <Label>Environment Type</Label>
                                <Select value={newEnvType} onValueChange={(value: any) => setNewEnvType(value)}>
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {environmentTypes.map((type) => (
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

                            <div className="space-y-2">
                                <Label>Description (Optional)</Label>
                                <Input
                                    placeholder="Brief description of this environment"
                                    value={newEnvDescription}
                                    onChange={(e) => setNewEnvDescription(e.target.value)}
                                />
                            </div>

                            <div className="flex justify-end gap-2">
                                <Button variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                                    Cancel
                                </Button>
                                <Button onClick={createEnvironment}>
                                    Create Environment
                                </Button>
                            </div>
                        </div>
                    </DialogContent>
                </Dialog>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Environment List */}
                <div className="lg:col-span-1 space-y-4">
                    <h2 className="font-semibold">Environments</h2>
                    {environments.map((env) => {
                        const typeConfig = getTypeConfig(env.type);
                        return (
                            <Card
                                key={env.id}
                                className={`p-4 cursor-pointer transition-all hover:shadow-lg ${selectedEnv?.id === env.id ? 'ring-2 ring-primary' : ''}`}
                                onClick={() => setSelectedEnv(env)}
                            >
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className={`w-3 h-3 rounded-full ${typeConfig.color}`}></div>
                                        <div>
                                            <h3 className="font-medium">{env.name}</h3>
                                            <p className="text-sm text-muted-foreground">{env.description}</p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <Badge variant="outline">
                                            {Object.keys(env.variables).length} vars
                                        </Badge>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                deleteEnvironment(env.id);
                                            }}
                                            className="text-destructive hover:text-destructive"
                                        >
                                            <Trash2 className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </div>
                            </Card>
                        );
                    })}

                    {environments.length === 0 && (
                        <Card className="p-8 text-center">
                            <Server className="h-8 w-8 mx-auto mb-3 text-muted-foreground" />
                            <h3 className="font-medium mb-2">No environments</h3>
                            <p className="text-sm text-muted-foreground">
                                Create your first environment to get started.
                            </p>
                        </Card>
                    )}
                </div>

                {/* Environment Details */}
                <div className="lg:col-span-2">
                    {selectedEnv ? (
                        <Card className="p-6">
                            <div className="space-y-6">
                                <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className={`w-4 h-4 rounded-full ${getTypeConfig(selectedEnv.type).color}`}></div>
                                        <div>
                                            <h2 className="text-xl font-semibold">{selectedEnv.name}</h2>
                                            <p className="text-muted-foreground">{selectedEnv.description}</p>
                                        </div>
                                    </div>
                                    <Badge variant="outline" className="flex items-center gap-1">
                                        {React.createElement(getTypeConfig(selectedEnv.type).icon, { className: "h-3 w-3" })}
                                        {getTypeConfig(selectedEnv.type).label}
                                    </Badge>
                                </div>

                                <Tabs defaultValue="variables" className="space-y-4">
                                    <TabsList>
                                        <TabsTrigger value="variables">Variables</TabsTrigger>
                                        <TabsTrigger value="endpoints">Endpoints</TabsTrigger>
                                        <TabsTrigger value="settings">Settings</TabsTrigger>
                                    </TabsList>

                                    <TabsContent value="variables" className="space-y-4">
                                        <div className="flex items-center justify-between">
                                            <h3 className="font-medium">Environment Variables</h3>
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                onClick={() => {
                                                    const key = prompt('Variable name:');
                                                    if (key) {
                                                        updateEnvironmentVariable(selectedEnv.id, key, '');
                                                    }
                                                }}
                                            >
                                                <Plus className="h-4 w-4 mr-1" />
                                                Add Variable
                                            </Button>
                                        </div>

                                        <div className="space-y-3">
                                            {Object.entries(selectedEnv.variables).map(([key, value]) => (
                                                <div key={key} className="flex items-center gap-2">
                                                    <Input
                                                        value={key}
                                                        readOnly
                                                        className="w-1/3 font-mono text-sm"
                                                    />
                                                    <Input
                                                        value={value}
                                                        onChange={(e) => updateEnvironmentVariable(selectedEnv.id, key, e.target.value)}
                                                        className="flex-1 font-mono text-sm"
                                                        type={key.toLowerCase().includes('password') || key.toLowerCase().includes('secret') ? 'password' : 'text'}
                                                    />
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={() => deleteEnvironmentVariable(selectedEnv.id, key)}
                                                        className="text-destructive"
                                                    >
                                                        <Trash2 className="h-4 w-4" />
                                                    </Button>
                                                </div>
                                            ))}

                                            {Object.keys(selectedEnv.variables).length === 0 && (
                                                <p className="text-muted-foreground text-center py-8">
                                                    No environment variables configured.
                                                </p>
                                            )}
                                        </div>
                                    </TabsContent>

                                    <TabsContent value="endpoints" className="space-y-4">
                                        <h3 className="font-medium">Service Endpoints</h3>
                                        <div className="space-y-4">
                                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                                <div className="space-y-2">
                                                    <Label>API Endpoint</Label>
                                                    <Input
                                                        value={selectedEnv.endpoints.api || ''}
                                                        onChange={(e) => {
                                                            setEnvironments(environments.map(env =>
                                                                env.id === selectedEnv.id
                                                                    ? { ...env, endpoints: { ...env.endpoints, api: e.target.value } }
                                                                    : env
                                                            ));
                                                            setSelectedEnv({ ...selectedEnv, endpoints: { ...selectedEnv.endpoints, api: e.target.value } });
                                                        }}
                                                        placeholder="https://api.example.com"
                                                    />
                                                </div>

                                                <div className="space-y-2">
                                                    <Label>Database</Label>
                                                    <Input
                                                        value={selectedEnv.endpoints.database || ''}
                                                        onChange={(e) => {
                                                            setEnvironments(environments.map(env =>
                                                                env.id === selectedEnv.id
                                                                    ? { ...env, endpoints: { ...env.endpoints, database: e.target.value } }
                                                                    : env
                                                            ));
                                                            setSelectedEnv({ ...selectedEnv, endpoints: { ...selectedEnv.endpoints, database: e.target.value } });
                                                        }}
                                                        placeholder="db.example.com:5432"
                                                    />
                                                </div>

                                                <div className="space-y-2">
                                                    <Label>Cache/Redis</Label>
                                                    <Input
                                                        value={selectedEnv.endpoints.cache || ''}
                                                        onChange={(e) => {
                                                            setEnvironments(environments.map(env =>
                                                                env.id === selectedEnv.id
                                                                    ? { ...env, endpoints: { ...env.endpoints, cache: e.target.value } }
                                                                    : env
                                                            ));
                                                            setSelectedEnv({ ...selectedEnv, endpoints: { ...selectedEnv.endpoints, cache: e.target.value } });
                                                        }}
                                                        placeholder="cache.example.com:6379"
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                    </TabsContent>

                                    <TabsContent value="settings" className="space-y-4">
                                        <h3 className="font-medium">Environment Settings</h3>
                                        <div className="space-y-4">
                                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                                <div className="space-y-2">
                                                    <Label>Environment Type</Label>
                                                    <Select
                                                        value={selectedEnv.type}
                                                        onValueChange={(value: any) => {
                                                            setEnvironments(environments.map(env =>
                                                                env.id === selectedEnv.id ? { ...env, type: value } : env
                                                            ));
                                                            setSelectedEnv({ ...selectedEnv, type: value });
                                                        }}
                                                    >
                                                        <SelectTrigger>
                                                            <SelectValue />
                                                        </SelectTrigger>
                                                        <SelectContent>
                                                            {environmentTypes.map((type) => (
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

                                                <div className="space-y-2">
                                                    <Label>Description</Label>
                                                    <Input
                                                        value={selectedEnv.description}
                                                        onChange={(e) => {
                                                            setEnvironments(environments.map(env =>
                                                                env.id === selectedEnv.id ? { ...env, description: e.target.value } : env
                                                            ));
                                                            setSelectedEnv({ ...selectedEnv, description: e.target.value });
                                                        }}
                                                        placeholder="Environment description"
                                                    />
                                                </div>
                                            </div>
                                        </div>
                                    </TabsContent>
                                </Tabs>
                            </div>
                        </Card>
                    ) : (
                        <Card className="p-12 text-center">
                            <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                            <h3 className="text-lg font-semibold mb-2">Select an environment</h3>
                            <p className="text-muted-foreground">
                                Choose an environment from the list to view and edit its configuration.
                            </p>
                        </Card>
                    )}
                </div>
            </div>
        </div>
    );
}
