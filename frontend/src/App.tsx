import { Routes, Navigate, Route } from "react-router-dom";
import "./App.css";
import MainDashboard from "./app/dashboard/dashboard";
import LoginPage from "./app/login/login";
import { useEffect, useState } from "react";

function App() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/auth/me", { credentials: "include" })
      .then(async (res) => {
        const body = await res.json();
        if (!res.ok) {
          throw new Error("unauthenticated");
        }
        setUser(body.data);
      })
      .catch(() => {
        setUser(null);
      })
      .finally(() => setLoading(false));
  }, []);

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
