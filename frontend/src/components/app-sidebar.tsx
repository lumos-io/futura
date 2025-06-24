import * as React from "react";
import {
  BookOpen,
  Frame,
  GalleryVerticalEnd,
  Map,
  NetworkIcon,
  PieChart,
  Store,
  ServerIcon,
  SquareTerminal,
} from "lucide-react";

import { NavMain } from "@/components/nav-main";
import { NavKubernetes } from "@/components/nav-projects";
import { NavUser } from "@/components/nav-user";
import { TeamSwitcher } from "@/components/team-switcher";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";

const data = {
  teams: [
    {
      name: "Futura",
      logo: GalleryVerticalEnd,
      plan: "Enterprise",
    },
  ],
  navMain: [
    {
      title: "Clusters",
      url: "#",
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: "Overview",
          url: "/dashboard/clusters/overview",
          isActive: true,
        },
        {
          title: "Workloads Health",
          url: "/dashboard/clusters/workloads-health",
          isActive: false,
        },
        {
          title: "Services",
          url: "/dashboard/clusters/services",
          isActive: false,
        },
        {
          title: "Jobs",
          url: "/dashboard/clusters/jobs",
          isActive: false,
        },
        {
          title: "Events",
          url: "/dashboard/clusters/events",
          isActive: false,
        },
      ],
    },
    {
      title: "Infrastructure",
      url: "#",
      icon: Map,
      items: [
        {
          title: "Health Overview",
          url: "/dashboard/infrastructure/health-overview",
          isActive: false,
        },
        {
          title: "Cost Optimization",
          url: "/dashboard/infrastructure/cost-optimization",
          isActive: false,
        },
        {
          title: "Vulnerabilities",
          url: "/dashboard/infrastructure/vulnerabilities",
          isActive: false,
        },
      ],
    },
    {
      title: "Access Management",
      url: "#",
      icon: BookOpen,
      items: [
        {
          title: "Users",
          url: "/dashboard/access-management/users",
          isActive: false,
        },
        {
          title: "Teams",
          url: "/dashboard/access-management/teams",
          isActive: false,
        },
        {
          title: "API Keys",
          url: "/dashboard/access-management/api-keys",
          isActive: false,
        },
        {
          title: "Audit Trail",
          url: "/dashboard/access-management/audit-trail",
          isActive: false,
        },
      ],
    },
  ],
  kubernetes: [
    {
      name: "Nodes",
      url: "/dashboard/kubernetes/nodes",
      icon: Frame,
      isActive: false,
    },
    {
      name: "Namespaces",
      url: "/dashboard/kubernetes/namespaces",
      icon: PieChart,
      isActive: false,
    },
    {
      name: "Workloads",
      url: "/dashboard/kubernetes/workloads",
      icon: ServerIcon,
      isActive: false,
    },
    {
      name: "Network",
      url: "/dashboard/kubernetes/network",
      icon: NetworkIcon,
      isActive: false,
    },
    {
      name: "Storage",
      url: "/dashboard/kubernetes/storage",
      icon: Store,
      isActive: false,
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <TeamSwitcher teams={data.teams} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavKubernetes kubernetes={data.kubernetes} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
