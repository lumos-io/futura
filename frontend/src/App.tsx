import { Routes, Navigate, Route } from "react-router-dom";
import "./App.css";
import MainDashboard from "./app/dashboard/dashboard";
import LoginPage from "./app/login/login";
import { useAuth } from "./hooks/auth_provider";

function App() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <Routes>
      {/* Public route: Login page */}
      <Route
        path="/login"
        element={!user ? <LoginPage /> : <Navigate to="/dashboard" replace />}
      />
      {/* Protected route: Dashboard */}
      <Route
        path="/dashboard"
        element={user ? <MainDashboard /> : <Navigate to="/login" replace />}
      />
      {/* Redirect unknown routes */}
      <Route
        path="*"
        element={<Navigate to={user ? "/dashboard" : "/login"} replace />}
      />
    </Routes>
  );
}

export default App;
