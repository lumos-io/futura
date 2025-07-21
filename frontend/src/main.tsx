import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { FlagProvider } from "@unleash/proxy-client-react";
import { AuthProvider } from "@/hooks/auth-provider.tsx";
import DevBanner from "@/components/banner/dev-banner.tsx";
import "./index.css";
import App from "./App.tsx";

const config = {
  url: import.meta.env.VITE_UNLEASH_URL,
  clientKey: import.meta.env.VITE_UNLEASH_FRONTEND_CLIENT_KEY,
  refreshInterval: 15,
  appName: import.meta.env.VITE_UNLEASH_APP_NAME,
};

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <BrowserRouter>
        <div
          style={{ display: "flex", flexDirection: "column", height: "100vh" }}
        >
          <FlagProvider config={config}>
            <DevBanner />
            <div style={{ flexGrow: 1, overflowY: "auto" }}>
              <App />
            </div>
          </FlagProvider>
        </div>
      </BrowserRouter>
    </AuthProvider>
  </StrictMode>
);
