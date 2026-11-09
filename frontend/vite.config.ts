import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// public/.gitkeep is copied into dist, so dist stays in git (empty) and the
// Go embed in embed.go builds before the first frontend build.
export default defineConfig({
  plugins: [svelte()],
});
