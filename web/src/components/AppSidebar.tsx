import { NavLink, useLocation } from "react-router-dom";
import { 
  GitBranch, 
  Settings, 
  Activity,
  FolderOpen,
  Plus,
  Layers,
  Clock,
  BarChart3
} from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";

const navigationItems = [
  { 
    title: "Pipelines", 
    url: "/", 
    icon: GitBranch,
    description: "Manage CI/CD pipelines"
  },
  { 
    title: "Jobs", 
    url: "/jobs", 
    icon: Activity,
    description: "View pipeline executions"
  },
  { 
    title: "History", 
    url: "/history", 
    icon: Clock,
    description: "Execution history"
  },
  { 
    title: "Analytics", 
    url: "/analytics", 
    icon: BarChart3,
    description: "Performance metrics"
  }
];

const managementItems = [
  { 
    title: "Environments", 
    url: "/environments", 
    icon: Layers,
    description: "Environment configs"
  },
  { 
    title: "Settings", 
    url: "/settings", 
    icon: Settings,
    description: "Platform settings"
  }
];

export function AppSidebar() {
  const { state } = useSidebar();
  const location = useLocation();
  const isCollapsed = state === "collapsed";
  
  const isActive = (path: string) => {
    if (path === "/") {
      return location.pathname === "/";
    }
    return location.pathname.startsWith(path);
  };

  const getNavClass = (path: string) => {
    return isActive(path) 
      ? "bg-accent text-accent-foreground border-r-2 border-primary" 
      : "hover:bg-muted/50 text-muted-foreground hover:text-foreground";
  };

  return (
    <Sidebar className={isCollapsed ? "w-16" : "w-64"} collapsible="icon">
      <SidebarContent className="gap-2">
        {/* Quick Actions */}
        {!isCollapsed && (
          <div className="px-3 py-2">
            <Button 
              asChild 
              variant="default" 
              className="w-full bg-gradient-to-r from-primary to-primary-hover hover:from-primary-hover hover:to-primary"
            >
              <NavLink to="/pipelines/new">
                <Plus className="h-4 w-4 mr-2" />
                New Pipeline
              </NavLink>
            </Button>
          </div>
        )}

        {/* Main Navigation */}
        <SidebarGroup>
          <SidebarGroupLabel>Platform</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navigationItems.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild className="h-12">
                    <NavLink 
                      to={item.url} 
                      className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-all ${getNavClass(item.url)}`}
                    >
                      <item.icon className="h-5 w-5 flex-shrink-0" />
                      {!isCollapsed && (
                        <div className="flex flex-col items-start min-w-0">
                          <span className="font-medium text-sm">{item.title}</span>
                          <span className="text-xs text-muted-foreground truncate">
                            {item.description}
                          </span>
                        </div>
                      )}
                    </NavLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Management */}
        <SidebarGroup>
          <SidebarGroupLabel>Management</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {managementItems.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild className="h-12">
                    <NavLink 
                      to={item.url} 
                      className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-all ${getNavClass(item.url)}`}
                    >
                      <item.icon className="h-5 w-5 flex-shrink-0" />
                      {!isCollapsed && (
                        <div className="flex flex-col items-start min-w-0">
                          <span className="font-medium text-sm">{item.title}</span>
                          <span className="text-xs text-muted-foreground truncate">
                            {item.description}
                          </span>
                        </div>
                      )}
                    </NavLink>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}