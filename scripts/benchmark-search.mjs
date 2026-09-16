import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { performance } from "node:perf_hooks";

export function percentile(samples, value) {
  if (!samples.length || value <= 0 || value > 1 || samples.some((sample) => !Number.isFinite(sample))) {
    throw new Error("percentile requires finite samples and a value in (0, 1]");
  }
  const sorted = samples.toSorted((a, b) => a - b);
  return sorted[Math.ceil(sorted.length * value) - 1];
}

export function serverTiming(header, name) {
	const match = new RegExp(`(?:^|,)\\s*${name};dur=([0-9.]+)`).exec(header ?? "");
	if (!match) throw new Error(`missing ${name} Server-Timing value`);
	return Number(match[1]);
}

async function run(baseURL) {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const evaluation = JSON.parse(await readFile(path.join(root, "docs/chapter-1/evaluation.json"), "utf8"));
  const mode = process.env.SEARCH_MODE ?? "hybrid";
  if (!["lexical", "semantic", "hybrid"].includes(mode)) throw new Error("SEARCH_MODE must be lexical, semantic, or hybrid");
  const samples = [];
  const retrieval = [];
  const model = [];

  for (let round = 0; round < 6; round++) {
    for (const { query } of evaluation) {
      const url = new URL("/api/v1/search", baseURL);
      url.searchParams.set("q", query);
      url.searchParams.set("page", "1");
      url.searchParams.set("limit", "5");
      url.searchParams.set("sort", "relevance");
      url.searchParams.set("mode", mode);
      const start = performance.now();
      const response = await fetch(url, { signal: AbortSignal.timeout(10_000) });
      if (!response.ok) throw new Error(`${url}: HTTP ${response.status}`);
      const body = await response.json();
      const elapsed = performance.now() - start;
      if (!Array.isArray(body.results)) throw new Error(`${url}: invalid search response`);
      if (round > 0) {
        samples.push(elapsed);
        retrieval.push(serverTiming(response.headers.get("server-timing"), "retrieval"));
        model.push(serverTiming(response.headers.get("server-timing"), "model"));
      }
    }
  }

  const result = {
    mode,
    queries: evaluation.length,
    measured_rounds: 5,
    samples: samples.length,
    end_to_end_p50_ms: Number(percentile(samples, 0.5).toFixed(2)),
    end_to_end_p95_ms: Number(percentile(samples, 0.95).toFixed(2)),
    retrieval_p50_ms: Number(percentile(retrieval, 0.5).toFixed(2)),
    retrieval_p95_ms: Number(percentile(retrieval, 0.95).toFixed(2)),
    model_p50_ms: Number(percentile(model, 0.5).toFixed(2)),
    model_p95_ms: Number(percentile(model, 0.95).toFixed(2)),
  };
  console.log(JSON.stringify(result, null, 2));
  if (result.retrieval_p95_ms >= 300) throw new Error(`retrieval p95 ${result.retrieval_p95_ms}ms must be below 300ms`);
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 3) {
    console.error("usage: node scripts/benchmark-search.mjs [api-base-url]");
    process.exitCode = 2;
  } else {
    run(process.argv[2] ?? "http://localhost:8080").catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
  }
}
