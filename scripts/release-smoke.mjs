import { writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

async function request(baseURL, pathname, options, fetchFn) {
  const response = await fetchFn(new URL(pathname, baseURL), { ...options, signal: AbortSignal.timeout(75_000) });
  if (!response.ok) throw new Error(`${pathname}: status ${response.status}`);
  return { status: response.status, body: await response.json() };
}

export async function smoke(baseURL, fetchFn = fetch) {
  const checks = [];
  const ready = await request(baseURL, "/readyz", undefined, fetchFn);
  if (ready.body.status !== "ready") throw new Error("readiness response is invalid");
  checks.push("ready");
  const lexical = await request(baseURL, "/api/v1/search?q=backpropagation&mode=lexical", undefined, fetchFn);
  if (!Array.isArray(lexical.body.results) || !lexical.body.results[0]?.uri) throw new Error("lexical response has no result");
  checks.push("lexical");
  for (const mode of ["semantic", "hybrid"]) {
    const result = await request(baseURL, `/api/v1/search?q=backpropagation&mode=${mode}`, undefined, fetchFn);
    if (!Array.isArray(result.body.results) || !result.body.results[0]?.uri) throw new Error(`${mode} response has no result`);
    checks.push(mode);
  }
  const related = await request(baseURL, `/api/v1/related?uri=${encodeURIComponent(lexical.body.results[0].uri)}`, undefined, fetchFn);
  if (!Array.isArray(related.body.results) || !related.body.results[0]?.uri) throw new Error("related response has no result");
  checks.push("related");
  const trends = await request(baseURL, "/api/v1/trends", undefined, fetchFn);
  if (typeof trends.body.total !== "number" || trends.body.total < 1) throw new Error("trends response is invalid");
  checks.push("trends");
  const synthesis = await request(baseURL, "/api/v1/answer", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ query: "peramalan curah hujan" }) }, fetchFn);
  if (synthesis.body.insufficient_evidence || !Array.isArray(synthesis.body.citations) || synthesis.body.citations.length === 0) throw new Error("synthesis response has no cited answer");
  checks.push("synthesis");
  return { generated_at: new Date().toISOString(), checks };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 4) {
    console.error("usage: node scripts/release-smoke.mjs [api-base-url] [output-file]");
    process.exitCode = 2;
  } else {
    const [baseURL = "http://localhost:8080", output] = process.argv.slice(2);
    smoke(baseURL).then(async result => {
      if (output) await writeFile(path.resolve(output), `${JSON.stringify(result, null, 2)}\n`);
      console.log(JSON.stringify(result));
    }).catch(error => {
      console.error(error.message);
      process.exitCode = 1;
    });
  }
}
