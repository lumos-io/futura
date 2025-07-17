// src/routes/dashboardRoutes.tsx
import { Navigate, RouteObject } from "react-router-dom";
import { useAuth } from "./hooks/auth_provider";

import MainDashboard from "@/app/dashboard/dashboard";
import LoginPage from "@/app/login/login";
import PrivateRoute from "@/components/private-route";
import ClustersOverview from "@/app/clusters/overview";
import ClusterWorkloadsHealth from "./app/clusters/workloads-health";
import ClusterEvents from "./app/clusters/events";
import ClusterJobs from "./app/clusters/jobs";
import InfrastructureHealthOverview from "./app/infrastructure/health-overview";
import CostOptimization from "./app/infrastructure/cost-optimization";
import Vulnerabilities from "./app/infrastructure/vulnerabilities";
import Users from "./app/access-management/users";
import Teams from "./app/access-management/teams";
import ApiKeys from "./app/access-management/api-keys";
import AuditTrail from "./app/access-management/audit-trail";
import KubernetesNodes from "./app/kubernetes/nodes";
import KubernetesNamespaces from "./app/kubernetes/namespaces";
import KubernetesNetwork from "./app/kubernetes/network";
import KubernetesStorage from "./app/kubernetes/storage";
import KubernetesWorkloads from "./app/kubernetes/workloads";
import CloudProviders from "./app/connect/cloud-providers";
import ClusterServices from "./app/clusters/services";

const allRoutes: RouteObject[] = [
  {
    path: "/login",
    element: <LoginRedirectWrapper />,
  },
  {
    path: "/dashboard",
    element: <MainDashboard />,
    children: [
      {
        index: true,
        element: <PrivateRoute element={<ClustersOverview />} />,
      },
      {
        path: "connect/cloud-providers",
        element: <PrivateRoute element={<CloudProviders />} />,
      },
      {
        path: "clusters/overview",
        element: <PrivateRoute element={<ClustersOverview />} />,
      },
      {
        path: "clusters/workloads-health",
        element: (
          <PrivateRoute
            element={<ClusterWorkloadsHealth title="Workloads Health" />}
          />
        ),
      },
      {
        path: "clusters/services",
        element: (
          <PrivateRoute
            element={<ClusterServices title="Cluster Services" />}
          />
        ),
      },
      {
        path: "clusters/events",
        element: (
          <PrivateRoute element={<ClusterEvents title="Cluster Events" />} />
        ),
      },
      {
        path: "clusters/jobs",
        element: (
          <PrivateRoute element={<ClusterJobs title="Cluster Jobs" />} />
        ),
      },

      {
        path: "infrastructure/health-overview",
        element: (
          <PrivateRoute
            element={
              <InfrastructureHealthOverview title="Infrastructure Health Overview" />
            }
          />
        ),
      },
      {
        path: "infrastructure/cost-optimization",
        element: (
          <PrivateRoute
            element={<CostOptimization title="Cost Optimization" />}
          />
        ),
      },
      {
        path: "infrastructure/vulnerabilities",
        element: (
          <PrivateRoute element={<Vulnerabilities title="Vulnerabilities" />} />
        ),
      },

      {
        path: "access-management/users",
        element: <PrivateRoute element={<Users title="Users Management" />} />,
      },
      {
        path: "access-management/teams",
        element: <PrivateRoute element={<Teams title="Teams Management" />} />,
      },
      {
        path: "access-management/api-keys",
        element: (
          <PrivateRoute element={<ApiKeys title="Api Keys Management" />} />
        ),
      },
      {
        path: "access-management/audit-trail",
        element: (
          <PrivateRoute
            element={<AuditTrail title="Audit Trail Management" />}
          />
        ),
      },

      {
        path: "kubernetes/nodes",
        element: (
          <PrivateRoute
            element={<KubernetesNodes title="Kubernetes Nodes" />}
          />
        ),
      },
      {
        path: "kubernetes/namespaces",
        element: (
          <PrivateRoute
            element={<KubernetesNamespaces title="Kubernetes Namespaces" />}
          />
        ),
      },
      {
        path: "kubernetes/workloads",
        element: (
          <PrivateRoute
            element={<KubernetesWorkloads title="Kubernetes Workloads" />}
          />
        ),
      },
      {
        path: "kubernetes/network",
        element: (
          <PrivateRoute
            element={<KubernetesNetwork title="Kubernetes Network" />}
          />
        ),
      },
      {
        path: "kubernetes/storage",
        element: (
          <PrivateRoute
            element={<KubernetesStorage title="Kubernetes Storage" />}
          />
        ),
      },
    ],
  },
  {
    path: "*",
    element: <WildcardRedirectWrapper />,
  },
];

export function LoginRedirectWrapper() {
  const { user } = useAuth();
  return user ? (
    <Navigate to="/dashboard/clusters/overview" replace />
  ) : (
    <LoginPage />
  );
}

export function WildcardRedirectWrapper() {
  const { user } = useAuth();
  return (
    <Navigate to={user ? "/dashboard/clusters/overview" : "/login"} replace />
  );
}

export default allRoutes;
