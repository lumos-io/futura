import path from "path"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@proto": path.resolve(__dirname, "../proto/gen"),
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: true,
    port: 8080,
  },
  base: "/",
  build: {
    outDir: "../backend/public", // Build frontend into the Go backend
    emptyOutDir: true,
  },
})
