import { createHash, randomBytes } from "node:crypto";
import { createReadStream, createWriteStream } from "node:fs";
import { access, mkdir, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";
import net from "node:net";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const backend = path.join(root, "apps/backend");
const compose = ["compose", "--env-file", ".env", "exec", "-T", "db"];

export function disposableDatabaseURL(value, database) {
  const url = new URL(value);
  if (url.protocol !== "postgres:" || url.hostname !== "127.0.0.1" || url.pathname === "/") throw new Error("DATABASE_URL must target a loopback PostgreSQL database");
  url.pathname = `/${database}`;
  return url.toString();
}

export function disposableNames(now = Date.now(), random = randomBytes(3).toString("hex")) {
  const suffix = `${now}_${random}`;
  return { build: `searchlens_phase3_build_${suffix}`, restore: `searchlens_phase3_restore_${suffix}` };
}

function execute(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: root, env: process.env, ...options });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", chunk => { stdout += chunk; });
    child.stderr.on("data", chunk => { stderr += chunk; });
    child.on("error", reject);
    child.on("close", code => code === 0 ? resolve(stdout.trim()) : reject(new Error(`${command} ${args.join(" ")}: ${stderr.trim() || `exit ${code}`}`)));
  });
}

function transfer(command, args, input, output) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: root, env: process.env, stdio: [input ? "pipe" : "ignore", output ? "pipe" : "ignore", "pipe"] });
    let stderr = "";
    child.stderr.on("data", chunk => { stderr += chunk; });
    if (input) createReadStream(input).pipe(child.stdin);
    if (output) child.stdout.pipe(createWriteStream(output));
    child.on("error", reject);
    child.on("close", code => code === 0 ? resolve() : reject(new Error(`${command} ${args.join(" ")}: ${stderr.trim() || `exit ${code}`}`)));
  });
}

async function hash(file) {
  const digest = createHash("sha256");
  for await (const chunk of createReadStream(file)) digest.update(chunk);
  return digest.digest("hex");
}

function parseCounts(output, fields) {
  const result = {};
  for (const field of fields) {
    const match = output.match(new RegExp(`${field}=(\\d+)`));
    if (!match) throw new Error(`missing ${field} in ${output}`);
    result[field] = Number(match[1]);
  }
  return result;
}

function parseReport(output) {
  const match = output.match(/^postgresql=(.+) pgvector=(\S+) documents=(\d+) embeddings=(\d+)$/);
  if (!match) throw new Error(`invalid database report: ${output}`);
  return { postgresql: match[1], pgvector: match[2], documents: Number(match[3]), embeddings: Number(match[4]) };
}

async function freePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => server.once("error", reject).listen(0, "127.0.0.1", resolve));
  const { port } = server.address();
  await new Promise(resolve => server.close(resolve));
  return port;
}

async function waitForReady(baseURL, api) {
  for (let attempt = 0; attempt < 60; attempt++) {
    if (api.exitCode !== null) throw new Error("restored API exited before readiness");
    try {
      const response = await fetch(new URL("/readyz", baseURL), { signal: AbortSignal.timeout(1_000) });
      if (response.ok) return;
    } catch {}
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  throw new Error("restored API did not become ready");
}

async function stop(api) {
  if (!api || api.exitCode !== null) return;
	process.kill(-api.pid, "SIGINT");
  await Promise.race([new Promise(resolve => api.once("close", resolve)), new Promise(resolve => setTimeout(resolve, 5_000))]);
	if (api.exitCode === null) process.kill(-api.pid, "SIGKILL");
}

async function smoke(baseURL) {
  const request = async (pathname, options) => {
    const response = await fetch(new URL(pathname, baseURL), { ...options, signal: AbortSignal.timeout(75_000) });
    if (!response.ok) throw new Error(`${pathname}: status ${response.status}`);
    return response.json();
  };
  const lexical = await request("/api/v1/search?q=backpropagation&mode=lexical");
  if (!Array.isArray(lexical.results) || !lexical.results[0]?.uri) throw new Error("lexical smoke returned no result");
  await request("/api/v1/search?q=backpropagation&mode=semantic");
  await request("/api/v1/search?q=backpropagation&mode=hybrid");
  await request(`/api/v1/related?uri=${encodeURIComponent(lexical.results[0].uri)}`);
  await request("/api/v1/trends");
  const answer = await request("/api/v1/answer", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ query: "peramalan curah hujan" }) });
  if (typeof answer.insufficient_evidence !== "boolean" || !Array.isArray(answer.citations)) throw new Error("synthesis smoke returned invalid response");
  return ["ready", "lexical", "semantic", "hybrid", "related", "trends", "synthesis"];
}

async function main() {
  const backupDirectory = process.env.BACKUP_DIR;
  if (!backupDirectory || !path.isAbsolute(backupDirectory)) throw new Error("BACKUP_DIR must be an absolute existing directory");
  await access(backupDirectory);
  for (const name of ["DATABASE_URL", "POSTGRES_USER", "OPENROUTER_API_KEY"]) if (!process.env[name]) throw new Error(`${name} is required`);
  const names = disposableNames();
  const buildURL = disposableDatabaseURL(process.env.DATABASE_URL, names.build);
  const restoreURL = disposableDatabaseURL(process.env.DATABASE_URL, names.restore);
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  const dump = path.join(backupDirectory, `searchlens-phase3-${stamp}.dump`);
  let api;
  try {
    await execute("docker", [...compose, "createdb", "--username", process.env.POSTGRES_USER, names.build]);
    const go = (args, databaseURL) => execute("go", args, { cwd: backend, env: { ...process.env, DATABASE_URL: databaseURL } });
    await go(["run", "./cmd/migrate", "-dir", "migrations"], buildURL);
    await go(["run", "./cmd/migrate", "-dir", "migrations"], buildURL);
    const firstImport = parseCounts(await go(["run", "./cmd/import", "-corpus", "../../docs/repository-data"], buildURL), ["inserted", "updated", "rejected", "total"]);
    const secondImport = parseCounts(await go(["run", "./cmd/import", "-corpus", "../../docs/repository-data"], buildURL), ["inserted", "updated", "rejected", "total"]);
    const firstEmbed = parseCounts(await go(["run", "./cmd/embed"], buildURL), ["embedded", "skipped", "failed", "total"]);
    const secondEmbed = parseCounts(await go(["run", "./cmd/embed"], buildURL), ["embedded", "skipped", "failed", "total"]);
    const buildReport = parseReport(await go(["run", "./cmd/migrate", "-report"], buildURL));
    if (firstImport.rejected || secondImport.inserted || secondImport.rejected || firstEmbed.failed || secondEmbed.embedded || secondEmbed.failed || buildReport.documents !== 2190 || buildReport.embeddings !== 2190) throw new Error("clean-build idempotency checks failed");
    await transfer("docker", [...compose, "pg_dump", "--username", process.env.POSTGRES_USER, "--format=custom", "--dbname", names.build], undefined, dump);
    await execute("docker", [...compose, "createdb", "--username", process.env.POSTGRES_USER, names.restore]);
    await transfer("docker", [...compose, "pg_restore", "--username", process.env.POSTGRES_USER, "--no-owner", "--dbname", names.restore], dump);
    const restoreReport = parseReport(await go(["run", "./cmd/migrate", "-report"], restoreURL));
    if (restoreReport.documents !== buildReport.documents || restoreReport.embeddings !== buildReport.embeddings) throw new Error("restored counts do not match source");
    const port = await freePort();
	api = spawn("go", ["run", "./cmd/api"], { cwd: backend, detached: true, env: { ...process.env, DATABASE_URL: restoreURL, PORT: String(port) }, stdio: "ignore" });
    const baseURL = `http://127.0.0.1:${port}`;
    await waitForReady(baseURL, api);
    const smokeChecks = await smoke(baseURL);
    const evidence = { generated_at: new Date().toISOString(), clean_build: { first_import: firstImport, second_import: secondImport, first_embed: firstEmbed, second_embed: secondEmbed, report: buildReport }, backup: { filename: path.basename(dump), sha256: await hash(dump) }, restore: { report: restoreReport, smoke_checks: smokeChecks } };
    await mkdir(path.join(root, "docs/chapter-3"), { recursive: true });
    await writeFile(path.join(root, "docs/chapter-3/database-operations.json"), `${JSON.stringify(evidence, null, 2)}\n`);
    console.log(JSON.stringify(evidence));
  } finally {
    await stop(api);
    for (const database of [names.restore, names.build]) {
      try { await execute("docker", [...compose, "dropdb", "--if-exists", "--username", process.env.POSTGRES_USER, database]); } catch (error) { console.error(error.message); }
    }
  }
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) main().catch(error => { console.error(error.message); process.exitCode = 1; });
