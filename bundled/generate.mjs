#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.dirname(fileURLToPath(import.meta.url));
const cli = path.join(root, "node_modules", "allure", "cli.js");

const result = spawnSync(process.execPath, [cli, ...process.argv.slice(2)], {
  cwd: root,
  encoding: "utf8",
  env: process.env,
});

if (result.stdout) {
  process.stdout.write(result.stdout);
}
if (result.stderr) {
  process.stderr.write(result.stderr);
}

process.exit(result.status ?? 1);
