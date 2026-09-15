import { createHash } from "node:crypto";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { aggregate, scoreQuery, validateEvaluation } from "./evaluate-search.mjs";
import { findCorpusFiles } from "./audit-corpus.mjs";

export const candidates = [
  { model: "openai/text-embedding-3-small", dimensions: 1536, input_usd_per_million: 0.02 },
  { model: "openai/text-embedding-3-large", dimensions: 1536, input_usd_per_million: 0.13 },
  { model: "google/gemini-embedding-2", dimensions: 1536, input_usd_per_million: 0.2 },
];

export function canonicalInput(record) {
  return [
    `Title: ${record.title ?? ""}`,
    `Authors: ${Array.isArray(record.authors) ? record.authors.join(", ") : ""}`,
    `Subjects: ${record.subjects ?? ""}`,
    `Division: ${record.divisions ?? ""}`,
    `Abstract: ${record.abstract ?? ""}`,
  ].join("\n");
}

const normalize = (vector) => {
  const magnitude = Math.sqrt(vector.reduce((sum, value) => sum + value * value, 0));
  if (!magnitude) throw new Error("embedding vector has zero magnitude");
  return vector.map((value) => value / magnitude);
};

const rankNormalized = (normalizedQuery, documents, limit) =>
  documents
    .map(({ uri, embedding }) => ({
      uri,
      score: embedding.reduce((sum, value, index) => sum + value * normalizedQuery[index], 0),
    }))
    .sort((a, b) => b.score - a.score || a.uri.localeCompare(b.uri))
    .slice(0, limit);

export function rankByCosine(query, documents, limit = 10) {
  return rankNormalized(normalize(query), documents.map(({ uri, embedding }) => ({ uri, embedding: normalize(embedding) })), limit);
}

export function selectWinner(results, lexicalNDCG) {
  const ordered = [...results].sort((a, b) => {
    const difference = b.subset_ndcg_at_10 - a.subset_ndcg_at_10;
    if (Math.abs(difference) >= 0.01) return difference;
    return a.estimated_cost_usd - b.estimated_cost_usd || a.p95_ms - b.p95_ms;
  });
  return {
    model: ordered[0].model,
    dimensions: ordered[0].dimensions,
    passed: ordered[0].subset_ndcg_at_10 > lexicalNDCG,
    rule: "highest conceptual+bilingual nDCG@10; differences below 0.01 use lower cost then p95",
  };
}

export function validateEmbeddingResponse(body, count, dimensions, model) {
  if (!Array.isArray(body.data) || body.data.length !== count) throw new Error(`${model}: invalid response count`);
  const vectors = body.data.toSorted((a, b) => a.index - b.index).map(({ embedding }) => embedding);
  if (vectors.some((vector) => !Array.isArray(vector) || vector.length !== dimensions || vector.some((value) => !Number.isFinite(value)))) {
    throw new Error(`${model}: invalid embedding dimensions`);
  }
  return vectors;
}

async function requestEmbeddings(apiKey, candidate, inputs) {
  const started = performance.now();
  const response = await fetch("https://openrouter.ai/api/v1/embeddings", {
    method: "POST",
    headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
    body: JSON.stringify({ model: candidate.model, dimensions: candidate.dimensions, input: inputs }),
    signal: AbortSignal.timeout(60_000),
  });
  if (!response.ok) throw new Error(`${candidate.model}: OpenRouter HTTP ${response.status}`);
  const body = await response.json();
  const vectors = validateEmbeddingResponse(body, inputs.length, candidate.dimensions, candidate.model);
  return { vectors, tokens: body.usage?.total_tokens ?? 0, elapsed_ms: performance.now() - started };
}

async function embedAll(apiKey, candidate, inputs) {
  const vectors = [];
  const latency = [];
  let tokens = 0;
  for (let index = 0; index < inputs.length; index += 32) {
    const response = await requestEmbeddings(apiKey, candidate, inputs.slice(index, index + 32));
    vectors.push(...response.vectors);
    tokens += response.tokens;
    latency.push(response.elapsed_ms);
  }
  return { vectors, tokens, latency };
}

async function readCorpus(root, quarantine) {
  const records = (await Promise.all((await findCorpusFiles(root)).map(async (file) => JSON.parse(await readFile(file, "utf8"))))).flat();
  const unique = new Map();
  for (const record of records) if (!quarantine.has(record.uri) && !unique.has(record.uri)) unique.set(record.uri, record);
  return [...unique.values()].sort((a, b) => a.uri.localeCompare(b.uri));
}

const subsetNDCG = (evaluation, scores) => {
  const subset = scores.filter((_, index) => ["conceptual", "bilingual"].includes(evaluation[index].intent));
  return aggregate(subset).ndcg_at_10;
};

async function benchmark(root, evaluation, baseline, apiKey, cacheRoot) {
  const quarantine = new Set(
    JSON.parse(await readFile(path.join(root, "docs/chapter-2/quarantine.json"), "utf8")).map(({ uri }) => uri),
  );
  const records = await readCorpus(path.join(root, "docs/repository-data"), quarantine);
  const digest = createHash("sha256")
    .update(records.map((record) => `${record.uri}\0${canonicalInput(record)}`).join("\0"))
    .digest("hex");
  await mkdir(cacheRoot, { recursive: true });
  const results = [];

  for (const candidate of candidates) {
    const cachePath = path.join(cacheRoot, `${candidate.model.replaceAll("/", "-")}-${candidate.dimensions}.json`);
    let cached;
    try {
      cached = JSON.parse(await readFile(cachePath, "utf8"));
      if (cached.digest !== digest || cached.queries !== evaluation.map(({ query }) => query).join("\0")) cached = null;
    } catch {
      cached = null;
    }
    if (!cached) {
      const documents = await embedAll(apiKey, candidate, records.map(canonicalInput));
      const queries = await embedAll(apiKey, candidate, evaluation.map(({ query }) => query));
      cached = {
        digest,
        queries: evaluation.map(({ query }) => query).join("\0"),
        document_vectors: documents.vectors,
        query_vectors: queries.vectors,
        tokens: documents.tokens + queries.tokens,
        latency: [...documents.latency, ...queries.latency],
      };
      await writeFile(cachePath, JSON.stringify(cached));
    }
    const documents = records.map((record, index) => ({ uri: record.uri, embedding: normalize(cached.document_vectors[index]) }));
    const scores = evaluation.map((item, index) =>
      scoreQuery(item.judgments, rankNormalized(normalize(cached.query_vectors[index]), documents, 10).map(({ uri }) => uri)),
    );
    results.push({
      model: candidate.model,
      dimensions: candidate.dimensions,
      metrics: aggregate(scores),
      subset_ndcg_at_10: subsetNDCG(evaluation, scores),
      tokens: cached.tokens,
      estimated_cost_usd: Number(((cached.tokens / 1_000_000) * candidate.input_usd_per_million).toFixed(6)),
      p95_ms: Number(cached.latency.toSorted((a, b) => a - b)[Math.ceil(cached.latency.length * 0.95) - 1].toFixed(2)),
    });
  }

  const lexicalSubset =
    ((baseline.by_intent.conceptual.ndcg_at_10 * baseline.by_intent.conceptual.queries) +
      (baseline.by_intent.bilingual.ndcg_at_10 * baseline.by_intent.bilingual.queries)) /
    (baseline.by_intent.conceptual.queries + baseline.by_intent.bilingual.queries);
  return {
    generated_at: new Date().toISOString(),
    provider: "OpenRouter",
    corpus_records: records.length,
    dimensions: 1536,
    lexical_subset_ndcg_at_10: Number(lexicalSubset.toFixed(4)),
    candidates: results,
    selection: selectWinner(results, lexicalSubset),
    generation: {
      model: "openai/gpt-5.6-luna",
      input_token_limit: 16000,
      output_token_limit: 800,
      timeout_seconds: 30,
      max_concurrency: 4,
      estimated_max_cost_usd: 0.00416,
      basis: "Repository metadata and available abstracts only.",
    },
    pricing_checked_at: new Date().toISOString().slice(0, 10),
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  const root = path.resolve(path.dirname(script), "..");
  const apiKey = process.env.OPENROUTER_API_KEY;
  if (!apiKey) {
    console.error("OPENROUTER_API_KEY is required");
    process.exitCode = 1;
  } else {
    const evaluationPath = path.resolve(process.argv[2] ?? path.join(root, "docs/chapter-2/evaluation.json"));
    const baselinePath = path.resolve(process.argv[3] ?? path.join(root, "docs/chapter-2/lexical-baseline.json"));
    const outputPath = path.resolve(process.argv[4] ?? path.join(root, "docs/chapter-2/embedding-benchmark.json"));
    Promise.all([readFile(evaluationPath, "utf8").then(JSON.parse), readFile(baselinePath, "utf8").then(JSON.parse)])
      .then(async ([evaluation, baseline]) => {
        validateEvaluation(evaluation);
        const result = await benchmark(root, evaluation, baseline, apiKey, path.join(root, ".cache/embeddings"));
        await writeFile(outputPath, `${JSON.stringify(result, null, 2)}\n`);
        console.log(JSON.stringify(result, null, 2));
        if (!result.selection.passed) process.exitCode = 1;
      })
      .catch((error) => {
        console.error(error.message);
        process.exitCode = 1;
      });
  }
}
