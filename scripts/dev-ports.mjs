#!/usr/bin/env node
// Picks a free port trio for a dev instance so several worktrees can each run the app
// at the same time. The ports are derived from the checkout's own path, so a worktree
// keeps the same URL across restarts (bookmarks and open tabs survive), and only walks
// forward when something else already holds one.
//
//   node scripts/dev-ports.mjs             prints DOZZLE_PORT=… VITE_PORT=… AGENT_PORT=…
//   node scripts/dev-ports.mjs --json      prints the same as JSON
//   node scripts/dev-ports.mjs pnpm dev    runs the command with those ports exported
//
// The plain `make dev` defaults (3100/5173/7007) are deliberately outside these ranges,
// so an auto-assigned instance never collides with someone's hand-started one.

import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { spawn } from "node:child_process";
import { execFileSync } from "node:child_process";

const RANGES = {
  DOZZLE_PORT: { start: 3200, size: 600 },
  VITE_PORT: { start: 5200, size: 600 },
  AGENT_PORT: { start: 7100, size: 300 },
};

// The git common dir is shared by every worktree, so the worktree's own root is what
// makes the name unique. Outside a checkout the cwd is answer enough.
function instanceKey() {
  try {
    return execFileSync("git", ["rev-parse", "--show-toplevel"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
  } catch {
    return process.cwd();
  }
}

// localhost resolves to both stacks and different servers pick different ones: vite
// listens on ::1, the Go server on 127.0.0.1. A port is only free when both are.
function isFree(port) {
  const check = (host) =>
    new Promise((resolve) => {
      const server = createServer();
      server.once("error", () => resolve(false));
      server.listen({ port, host, exclusive: true }, () => server.close(() => resolve(true)));
    });
  return Promise.all([check("127.0.0.1"), check("::1")]).then(([a, b]) => a && b);
}

async function pick(name, seed) {
  const { start, size } = RANGES[name];
  const offset = parseInt(createHash("sha256").update(`${seed}:${name}`).digest("hex").slice(0, 8), 16) % size;
  for (let i = 0; i < size; i++) {
    const port = start + ((offset + i) % size);
    if (await isFree(port)) return port;
  }
  throw new Error(`no free port for ${name} in ${start}-${start + size - 1}`);
}

const seed = instanceKey();
const ports = {};
for (const name of Object.keys(RANGES)) ports[name] = await pick(name, seed);

const [, , ...argv] = process.argv;
const command = argv.filter((arg) => arg !== "--json");

if (command.length === 0) {
  if (argv.includes("--json")) {
    console.log(JSON.stringify({ ...ports, url: `http://localhost:${ports.DOZZLE_PORT}` }, null, 2));
  } else {
    for (const [name, port] of Object.entries(ports)) console.log(`${name}=${port}`);
  }
  process.exit(0);
}

console.log(
  `▸ dozzle dev on http://localhost:${ports.DOZZLE_PORT} (vite ${ports.VITE_PORT}, agent ${ports.AGENT_PORT})`,
);
const child = spawn(command[0], command.slice(1), {
  stdio: "inherit",
  env: { ...process.env, ...Object.fromEntries(Object.entries(ports).map(([k, v]) => [k, String(v)])) },
});
child.on("exit", (code, signal) => process.exit(signal ? 1 : (code ?? 0)));
