import React, {
  createContext,
  useState,
  useEffect,
  useContext,
  useCallback,
} from "react";

// Define the User type based on your backend's user model
type User = {
  id: number;
  name: string;
  email: string;
  avatar: string;
  organizationId: number;
};

// Define the shape of the AuthContext
type AuthContextType = {
  user: User | null;
  loading: boolean;
  login: (provider: string) => void;
  isAuthenticated: () => boolean;
  logout: () => void;
  refresh: () => Promise<void>;
};

// Create the AuthContext
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// AuthProvider component to wrap your application
export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // Define a constant for the sessionStorage key
  const EXPLICIT_LOGOUT_KEY = "explicitlyLoggedOut";

  /**
   * Handles refreshing the access token using the backend's /auth/refresh endpoint.
   * This function is called when an access token is likely expired.
   * It relies on the browser automatically sending the refresh_token cookie
   * to the /auth/refresh path.
   */
  const refresh = useCallback(async () => {
    try {
      const res = await fetch("/auth/refresh", {
        method: "POST",
        credentials: "include", // Ensure cookies are sent
      });

      if (!res.ok) {
        // If refresh fails, it means the refresh token is invalid or expired.
        // The user is no longer authenticated.
        throw new Error("Refresh failed");
      }
      // If refresh is successful, new access_token and refresh_token cookies
      // have been set by the backend.
    } catch (error) {
      console.error("Error during token refresh:", error);
      // Clear user state if refresh fails, indicating full logout
      setUser(null);
      throw error; // Re-throw to propagate the error
    }
  }, []); // No dependencies, as it only interacts with a fixed endpoint

  /**
   * Fetches the current user's information from the backend.
   * This function attempts to get user data, and if it receives a 401 (Unauthorized),
   * it tries to refresh the token and then retries fetching user data.
   */
  const fetchUser = useCallback(async () => {
    setLoading(true); // Start loading state

    try {
      let res = await fetch("/auth/me", {
        credentials: "include", // Ensure cookies are sent
      });

      if (res.status === 401) {
        // Access token is expired or missing. Try to refresh.
        try {
          await refresh(); // Attempt to refresh the token
          // If refresh succeeds, new cookies are set. Retry fetching user data.
          res = await fetch("/auth/me", {
            credentials: "include",
          });
        } catch (refreshError) {
          // If refresh itself failed, or the second /auth/me call fails,
          // the user is definitively not authenticated.
          console.error(
            "Authentication failed after refresh attempt:",
            refreshError
          );
          setUser(null);
          setLoading(false);
          return; // Exit early as user is not authenticated
        }
      }

      if (!res.ok) {
        // If, after potential refresh, the response is still not OK (e.g., 401, 500),
        // it means the user is not authenticated.
        throw new Error("Not authenticated or server error");
      }

      const body = await res.json();
      setUser(body.data); // Set the user data
    } catch (error) {
      console.error("Error fetching user:", error);
      setUser(null); // Clear user state on any fetch error
    } finally {
      setLoading(false); // End loading state
    }
  }, [refresh]); // fetchUser depends on the refresh function

  /**
   * Logs the user out by calling the backend's /auth/logout endpoint.
   * Clears user state and redirects to the login page.
   */
  const logout = useCallback(async () => {
    setLoading(true); // Indicate loading while logging out
    setUser(null); // Immediately clear user state

    // Set a flag in sessionStorage to indicate explicit logout
    sessionStorage.setItem(EXPLICIT_LOGOUT_KEY, "true");

    try {
      await fetch("/auth/logout", {
        method: "POST",
        credentials: "include", // Ensure cookies are sent for logout
      });
    } catch (error) {
      console.error("Error during logout:", error);
      // Even if logout fetch fails, we still want to redirect
    } finally {
      // Redirect to login page after logout attempt
      window.location.href = "/login";
    }
  }, [EXPLICIT_LOGOUT_KEY]); // Dependency on the key constant

  /**
   * Initiates the OAuth login process for a given provider (e.g., "github", "google").
   * This redirects the user to the backend's OAuth handler.
   */
  const login = useCallback(
    (provider: string) => {
      // Clear the explicit logout flag if a login attempt is made
      sessionStorage.removeItem(EXPLICIT_LOGOUT_KEY);
      window.location.href = `/auth/${provider}/login`;
    },
    [EXPLICIT_LOGOUT_KEY]
  ); // Dependency on the key constant

  /**
   * Checks if the user is currently authenticated.
   * This is determined by the presence of user data in the state.
   */
  const isAuthenticated = useCallback(() => {
    return user !== null;
  }, [user]); // Depends on the user state

  // Effect to handle initial session check on component mount
  useEffect(() => {
    const isLoginPage = window.location.pathname === "/login";
    const wasExplicitlyLoggedOut =
      sessionStorage.getItem(EXPLICIT_LOGOUT_KEY) === "true";

    if (isLoginPage && wasExplicitlyLoggedOut) {
      // If on the login page AND user explicitly logged out in this session,
      // assume not authenticated and avoid the network call.
      setLoading(false);
    } else {
      // Otherwise, always attempt to fetch user data to verify session status.
      fetchUser();
    }
  }, [fetchUser, EXPLICIT_LOGOUT_KEY]); // Dependencies on fetchUser and the key constant

  // Effect to handle redirection if user is authenticated and on the login page
  useEffect(() => {
    // Only proceed if loading is complete and user status is determined
    if (!loading) {
      const isLoginPage = window.location.pathname === "/login";
      if (user && isLoginPage) {
        // User is authenticated AND on the login page, redirect them.
        // You can choose a default authenticated route, e.g., '/dashboard' or '/'
        window.location.href = "/"; // Redirect to home or dashboard
      }
    }
  }, [loading, user]); // Depends on loading and user state

  return (
    <AuthContext.Provider
      value={{ user, loading, isAuthenticated, login, logout, refresh }}
    >
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook for easier access to the AuthContext
export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
};
