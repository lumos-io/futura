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
  Blocks,
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
      title: "Connect",
      url: "#",
      icon: Blocks,
      isActive: false,
      items: [
        {
          title: "Cloud Providers",
          url: "/dashboard/connect/cloud-providers",
        },
      ],
    },
    {
      title: "Clusters",
      url: "#",
      icon: SquareTerminal,
      isActive: true,
      items: [
        {
          title: "Overview",
          url: "/dashboard/clusters/overview",
        },
        {
          title: "Nodes",
          url: "/dashboard/clusters/nodes",
        },
        {
          title: "Services",
          url: "/dashboard/clusters/services",
        },
        {
          title: "Events",
          url: "/dashboard/clusters/events",
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
        },
        {
          title: "Cost Optimization",
          url: "/dashboard/infrastructure/cost-optimization",
        },
        {
          title: "Vulnerabilities",
          url: "/dashboard/infrastructure/vulnerabilities",
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
        },
        {
          title: "Teams",
          url: "/dashboard/access-management/teams",
        },
        {
          title: "API Keys",
          url: "/dashboard/access-management/api-keys",
        },
        {
          title: "Audit Trail",
          url: "/dashboard/access-management/audit-trail",
        },
      ],
    },
  ],
  kubernetes: [
    {
      name: "Nodes",
      url: "/dashboard/kubernetes/nodes",
      icon: Frame,
    },
    {
      name: "Namespaces",
      url: "/dashboard/kubernetes/namespaces",
      icon: PieChart,
    },
    {
      name: "Workloads",
      url: "/dashboard/kubernetes/workloads",
      icon: ServerIcon,
    },
    {
      name: "Network",
      url: "/dashboard/kubernetes/network",
      icon: NetworkIcon,
    },
    {
      name: "Storage",
      url: "/dashboard/kubernetes/storage",
      icon: Store,
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
