import React, { createContext, useState, useEffect, useContext } from "react";

type User = {
  id: number;
  name: string;
  email: string;
  avatar: string;
  organizationId: number;
};

type AuthContextType = {
  user: User | null;
  loading: boolean;
  login: (provider: string) => void;
  isAuthenticated: () => boolean;
  logout: () => void;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchUser = async () => {
    try {
      let res = await fetch("/auth/me", {
        credentials: "include",
      });

      if (res.status === 401) {
        try {
          // refresh token
          await refresh();

          res = await fetch("/auth/me", {
            credentials: "include",
          });
        } catch {
          localStorage.removeItem("isAuthenticated");
          throw new Error("Unable to refresh");
        }
      }

      if (!res.ok) {
        throw new Error("Not authenticated");
      }

      const body = await res.json();
      setUser(body.data);
    } catch {
      setUser(null);
      localStorage.removeItem("isAuthenticated");
    } finally {
      setLoading(false);
    }
  };

  const refresh = async () => {
    const res = await fetch("/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) {
      localStorage.removeItem("isAuthenticated");
      throw new Error("Refresh failed");
    }
  };

  const logout = async () => {
    // Immediately disable any current rehydration attempt
    setLoading(true); // Block further logic
    setUser(null); // Clear state

    localStorage.removeItem("isAuthenticated");

    await fetch("/auth/logout", {
      method: "POST",
      credentials: "include",
    });

    window.location.href = "/login";
  };

  const login = (provider: string) => {
    console.log("login function called");
    // This just redirects to the backend OAuth handler
    window.location.href = `/auth/${provider}/login`;
  };

  const isAuthenticated = () => {
    const res = localStorage.getItem("isAuthenticated");
    if (res == "true") {
      return true;
    }
    return false;
  };

  // Set isAuthenticated on mount if it's a fresh login
  useEffect(() => {
    fetchUser();
  }, []);

  useEffect(() => {
    if (user) {
      localStorage.setItem("isAuthenticated", "true");
    }
  }, [user]);

  return (
    <AuthContext.Provider
      value={{ user, loading, isAuthenticated, login, logout, refresh }}
    >
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook for easier access
export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
};
