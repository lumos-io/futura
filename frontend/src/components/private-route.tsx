import { Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/auth_provider";
import { JSX } from "react";

interface PrivateRouteProps {
  element: JSX.Element;
}

export default function PrivateRoute({ element }: PrivateRouteProps) {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return <div>Loading...</div>;
  }

  return isAuthenticated() ? element : <Navigate to="/login" replace />;
}
