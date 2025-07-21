import { $ } from "bun";

const commitSha = await $`git rev-parse --short HEAD`.text();
process.env.VITE_COMMIT_SHA = commitSha.trim();

const buildTime = new Date().toISOString();
process.env.VITE_BUILD_TIME = buildTime;
process.env.VITE_ENV = process.env.NODE_ENV || "development";

await $`tsc -b`;
await $`vite build`;
