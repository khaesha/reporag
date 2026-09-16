import { createHash } from "node:crypto";
import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { percentile } from "./benchmark-search.mjs";

const relevant = (judgment) => judgment.relevance >= 2;
const mean = (values) => values.reduce((sum, value) => sum + value, 0) / values.length;
const rounded = (value) => Number(value.toFixed(4));

export function scoreQuery(judgments, rankedURIs, limit = 10) {
  const grades = new Map(judgments.map(({ uri, relevance }) => [uri, relevance]));
  const relevantURIs = judgments.filter(relevant).map(({ uri }) => uri);
  const ranked = rankedURIs.slice(0, limit);
  const found = relevantURIs.filter((uri) => ranked.includes(uri)).length;
  const first = ranked.findIndex((uri) => (grades.get(uri) ?? 0) >= 2);
  const dcg = ranked.reduce((sum, uri, index) => sum + ((2 ** (grades.get(uri) ?? 0)) - 1) / Math.log2(index + 2), 0);
  const ideal = judgments
    .map(({ relevance }) => relevance)
    .sort((a, b) => b - a)
    .slice(0, limit)
    .reduce((sum, grade, index) => sum + (2 ** grade - 1) / Math.log2(index + 2), 0);
  return {
    recall_at_10: relevantURIs.length ? found / relevantURIs.length : 0,
    mrr_at_10: first < 0 ? 0 : 1 / (first + 1),
    ndcg_at_10: ideal ? dcg / ideal : 0,
    top_5_success: ranked.slice(0, 5).some((uri) => (grades.get(uri) ?? 0) >= 2) ? 1 : 0,
  };
}

export function aggregate(entries) {
  const fields = ["recall_at_10", "mrr_at_10", "ndcg_at_10", "top_5_success"];
  return Object.fromEntries(fields.map((field) => [field, rounded(mean(entries.map((entry) => entry[field])))]));
}

export function validateEvaluation(evaluation) {
  if (!Array.isArray(evaluation) || evaluation.length < 100) throw new Error("evaluation must contain at least 100 queries");
  const ids = new Set();
  const queries = new Set();
  for (const item of evaluation) {
    if (typeof item.id !== "string" || !item.id || ids.has(item.id)) throw new Error(`invalid or duplicate id: ${item.id}`);
    if (typeof item.query !== "string" || !item.query.trim() || [...item.query].length > 200 || queries.has(item.query)) {
      throw new Error(`invalid or duplicate query: ${item.query}`);
    }
    if (!["exact_title", "exact_author", "lexical", "conceptual", "bilingual", "abbreviation"].includes(item.intent)) {
      throw new Error(`${item.id}: invalid intent`);
    }
    if (typeof item.division !== "string" || !item.division) throw new Error(`${item.id}: division is required`);
    if (!Array.isArray(item.judgments) || !item.judgments.some(relevant)) throw new Error(`${item.id}: relevant judgment is required`);
    const judgedURIs = new Set();
    for (const judgment of item.judgments) {
      if (typeof judgment.uri !== "string" || ![1, 2, 3].includes(judgment.relevance)) throw new Error(`${item.id}: invalid judgment`);
      if (judgedURIs.has(judgment.uri)) throw new Error(`${item.id}: duplicate judgment URI`);
      judgedURIs.add(judgment.uri);
    }
    ids.add(item.id);
    queries.add(item.query);
  }
}

const grouped = (evaluation, scores, key) => {
  const groups = new Map();
  evaluation.forEach((item, index) => {
    const entries = groups.get(item[key]) ?? [];
    entries.push(scores[index]);
    groups.set(item[key], entries);
  });
  return Object.fromEntries([...groups].sort().map(([name, entries]) => [name, { queries: entries.length, ...aggregate(entries) }]));
};

async function run(baseURL, evaluationPath, mode) {
	if (!['lexical', 'semantic', 'hybrid'].includes(mode)) throw new Error('mode must be lexical, semantic, or hybrid');
  const contents = await readFile(evaluationPath);
  const evaluation = JSON.parse(contents);
  validateEvaluation(evaluation);
  const latency = [];
  const rankings = [];

  for (let round = 0; round < 6; round++) {
    for (const [index, item] of evaluation.entries()) {
      const url = new URL("/api/v1/search", baseURL);
      url.searchParams.set("q", item.query);
      url.searchParams.set("page", "1");
      url.searchParams.set("limit", "10");
      url.searchParams.set("sort", "relevance");
		url.searchParams.set("mode", mode);
      const start = performance.now();
      const response = await fetch(url, { signal: AbortSignal.timeout(40_000) });
      if (!response.ok) throw new Error(`${url}: HTTP ${response.status}`);
      const body = await response.json();
      if (!Array.isArray(body.results)) throw new Error(`${url}: invalid search response`);
      if (round > 0) latency.push(performance.now() - start);
      if (round === 1) rankings[index] = body.results.map(({ uri }) => uri);
    }
  }

  const scores = evaluation.map((item, index) => scoreQuery(item.judgments, rankings[index]));
  return {
    generated_at: new Date().toISOString(),
    corpus_api: baseURL,
		mode,
    evaluation_sha256: createHash("sha256").update(contents).digest("hex"),
    queries: evaluation.length,
    metrics: aggregate(scores),
    by_intent: grouped(evaluation, scores, "intent"),
    by_division: grouped(evaluation, scores, "division"),
    latency: {
      measured_rounds: 5,
      samples: latency.length,
      p50_ms: Number(percentile(latency, 0.5).toFixed(2)),
      p95_ms: Number(percentile(latency, 0.95).toFixed(2)),
      max_ms: Number(Math.max(...latency).toFixed(2)),
    },
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 6) {
    console.error("usage: node scripts/evaluate-search.mjs [api-base-url] [evaluation-file] [output-file] [mode]");
    process.exitCode = 2;
  } else {
    const root = path.resolve(path.dirname(script), "..");
    const baseURL = process.argv[2] ?? "http://localhost:8080";
    const evaluationPath = path.resolve(process.argv[3] ?? path.join(root, "docs/chapter-2/evaluation.json"));
    const mode = process.argv[5] ?? "lexical";
    run(baseURL, evaluationPath, mode)
      .then(async (result) => {
        const output = `${JSON.stringify(result, null, 2)}\n`;
        if (process.argv[4]) await writeFile(path.resolve(process.argv[4]), output);
        console.log(output.trimEnd());
      })
      .catch((error) => {
        console.error(error.message);
        process.exitCode = 1;
      });
  }
}
