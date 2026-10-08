import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { performance } from "node:perf_hooks";

import { percentile, serverTiming } from "./benchmark-search.mjs";

async function measure(baseURL, endpoint, validate, samples, fetchFn) {
  const elapsed = [];
  const retrieval = [];
  for (let round = 0; round <= samples; round++) {
    const url = new URL(endpoint, baseURL);
    const started = performance.now();
    const response = await fetchFn(url, { signal: AbortSignal.timeout(10_000) });
    const body = await response.json();
    if (!response.ok || !validate(body)) throw new Error(`${url}: invalid response`);
    if (round > 0) {
      elapsed.push(performance.now() - started);
      retrieval.push(serverTiming(response.headers.get("server-timing"), "retrieval"));
    }
  }
  return {
    samples,
    end_to_end_p50_ms: Number(percentile(elapsed, 0.5).toFixed(2)),
    end_to_end_p95_ms: Number(percentile(elapsed, 0.95).toFixed(2)),
    retrieval_p50_ms: Number(percentile(retrieval, 0.5).toFixed(2)),
    retrieval_p95_ms: Number(percentile(retrieval, 0.95).toFixed(2)),
  };
}

export async function benchmark(baseURL, uri, samples = 100, fetchFn = fetch) {
  return {
    related: await measure(baseURL, `/api/v1/related?uri=${encodeURIComponent(uri)}&limit=6`, (body) => Array.isArray(body.results), samples, fetchFn),
    trends: await measure(baseURL, "/api/v1/trends", (body) => Number.isInteger(body.total), samples, fetchFn),
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 4) {
    console.error("usage: node scripts/benchmark-related-trends.mjs [api-base-url] [output-file]");
    process.exitCode = 2;
  } else {
    const root = path.resolve(path.dirname(script), "..");
    const baseURL = process.argv[2] ?? "http://localhost:8080";
    const outputPath = process.argv[3];
    const evaluation = JSON.parse(await readFile(path.join(root, "docs/chapter-2/evaluation.json"), "utf8"));
    const uri = evaluation[0]?.judgments?.[0]?.uri;
    if (typeof uri !== "string" || !uri) throw new Error("evaluation lacks a benchmark URI");
    benchmark(baseURL, uri).then(async (result) => {
      const output = `${JSON.stringify({ generated_at: new Date().toISOString(), corpus_api: baseURL, source_uri: uri, ...result }, null, 2)}\n`;
      if (outputPath) await writeFile(path.resolve(outputPath), output);
      console.log(output.trimEnd());
    }).catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
  }
}
