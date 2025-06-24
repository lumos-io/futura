import { Routes, Navigate, Route } from "react-router-dom";
import "./App.css";
import MainDashboard from "@/app/dashboard/dashboard";
import LoginPage from "@/app/login/login";
import { useAuth } from "@/hooks/auth_provider";
import PrivateRoute from "@/components/private-route";
import ClustersOverview from "@/app/clusters/overview";
import WorkloadsHealth from "./app/clusters/workloads-health";

function App() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <Routes>
      <Route
        path="/login"
        element={
          !user ? (
            <LoginPage />
          ) : (
            <Navigate to="/dashboard/clusters/overview" replace />
          )
        }
      />
      <Route path="/dashboard" element={<MainDashboard />}>
        <Route
          path="clusters/overview"
          element={
            <PrivateRoute
              element={<ClustersOverview title="Cluster Overview" />}
            />
          }
        />
        <Route
          path="clusters/workloads-health"
          element={
            <PrivateRoute
              element={<WorkloadsHealth title="Workloads Health" />}
            />
          }
        />
        <Route
          index
          element={
            <PrivateRoute
              element={<ClustersOverview title="Cluster Overview" />}
            />
          }
        />
      </Route>
      <Route
        path="*"
        element={
          <Navigate
            to={user ? "/dashboard/clusters/overview" : "/login"}
            replace
          />
        }
      />
    </Routes>
  );
}

export default App;
