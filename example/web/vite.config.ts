import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import fs from "fs";

// In Docker, ts source is mounted at /ts-src. Locally, it's at ../../ts/src.
const tsBase = fs.existsSync("/ts-src/index.ts")
  ? "/ts-src"
  : path.resolve(__dirname, "../../ts/src");

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "pixel-id/react": path.join(tsBase, "react/index.ts"),
      "pixel-id": path.join(tsBase, "index.ts"),
    },
  },
  server: {
    host: "0.0.0.0",
    port: 4200,
    watch: { usePolling: true },
    fs: {
      allow: ["/ts-src", "../.."],
    },
    proxy: {
      "/api": "http://api:8080",
    },
  },
});
