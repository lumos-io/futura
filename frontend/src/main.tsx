import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider } from "@/hooks/auth_provider";
import DevBanner from "@/components/banner/dev-banner.tsx";
import "./index.css";
import App from "./App.tsx";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <BrowserRouter>
        <div
          style={{ display: "flex", flexDirection: "column", height: "100vh" }}
        >
          <DevBanner />
          <div style={{ flexGrow: 1, overflowY: "auto" }}>
            <App />
          </div>
        </div>
      </BrowserRouter>
    </AuthProvider>
  </StrictMode>
);
