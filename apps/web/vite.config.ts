/// <reference types="vitest/config" />
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// Porta padrão do front: 5180 (5173 já está em uso nesta máquina). Em Docker o
// container escuta 5173 e o Compose publica em WEB_PORT.
const port = Number(process.env.WEB_PORT ?? 5180);
const apiTarget = process.env.API_PROXY_TARGET ?? "http://127.0.0.1:8480";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port,
    strictPort: true,
    proxy: { "/api": { target: apiTarget, changeOrigin: false } },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    css: false,
    exclude: ["**/node_modules/**", "**/dist/**", "e2e/**"],
  },
});
