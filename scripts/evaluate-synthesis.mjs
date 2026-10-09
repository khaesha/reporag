import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { percentile } from "./benchmark-search.mjs";
import { records } from "./verify-exact-titles.mjs";

export function selectPrompts(evaluation) {
  const prompts = [];
  for (const intent of ["conceptual", "bilingual", "abbreviation"]) {
    const groups = new Map();
    for (const item of evaluation.filter(item => item.intent === intent)) groups.set(item.division, [...(groups.get(item.division) ?? []), item]);
    const divisions = [...groups.keys()].sort();
    for (let index = 0; prompts.length < (intent === "conceptual" ? 8 : intent === "bilingual" ? 16 : 24); index++) {
      const item = groups.get(divisions[index % divisions.length])[Math.floor(index / divisions.length)];
      if (item) prompts.push({ id: item.id, query: item.query, expected: "supported" });
    }
  }
  return prompts;
}

export async function run(baseURL, prompts, corpus, fetchFn = fetch, onResult) {
  const byURI = new Map(corpus.map(item => [item.uri, item]));
  const results = [];
  for (const prompt of prompts) {
    const response = await fetchFn(new URL("/api/v1/answer", baseURL), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ query: prompt.query }), signal: AbortSignal.timeout(75_000) });
    const body = await response.json();
    const citations = (body.citations ?? []).map(citation => ({ ...citation, corpus_match: byURI.get(citation.uri)?.title === citation.title, abstract_available: Boolean(byURI.get(citation.uri)?.abstract?.trim()) }));
    const result = { ...prompt, request_id: response.headers.get("x-request-id"), status: response.status, answer: body.answer ?? "", insufficient_evidence: body.insufficient_evidence === true, citations, citation_valid: citations.every(item => item.corpus_match), telemetry: body.telemetry ?? null, claims: [] };
    results.push(result);
    if (onResult) await onResult(result);
  }
  return results;
}

export function summarize(prompts) {
  const telemetry = prompts.map(prompt => prompt.telemetry).filter(Boolean);
  const latency = telemetry.map(item => item.latency_ms).filter(Number.isFinite);
  const modelLatency = telemetry.map(item => item.model_latency_ms).filter(Number.isFinite);
  return {
    prompts: prompts.length,
    successful: prompts.filter(prompt => prompt.status >= 200 && prompt.status < 300).length,
    citations_valid: prompts.every(prompt => prompt.citation_valid),
    models: [...new Set(telemetry.map(item => item.model).filter(Boolean))].sort(),
    prompt_tokens: telemetry.reduce((total, item) => total + (item.prompt_tokens ?? 0), 0),
    completion_tokens: telemetry.reduce((total, item) => total + (item.completion_tokens ?? 0), 0),
    estimated_cost_usd: Number(telemetry.reduce((total, item) => total + (item.estimated_cost_usd ?? 0), 0).toFixed(8)),
    latency_ms: latency.length ? { p50: Number(percentile(latency, 0.5).toFixed(2)), p95: Number(percentile(latency, 0.95).toFixed(2)), max: Number(Math.max(...latency).toFixed(2)) } : null,
    model_latency_ms: modelLatency.length ? { p50: Number(percentile(modelLatency, 0.5).toFixed(2)), p95: Number(percentile(modelLatency, 0.95).toFixed(2)), max: Number(Math.max(...modelLatency).toFixed(2)) } : null,
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 5 || (process.argv[4] && process.argv[4] !== "--resume")) {
    console.error("usage: node scripts/evaluate-synthesis.mjs [api-base-url] [output-file] [--resume]");
    process.exitCode = 2;
  }
  const root = path.resolve(path.dirname(script), "..");
  const [baseURL = "http://localhost:8080", output = path.join(root, "docs/chapter-2/synthesis-results.json"), resume] = process.argv.slice(2);
  const [evaluation, corpus] = await Promise.all([readFile(path.join(root, "docs/chapter-2/evaluation.json"), "utf8").then(JSON.parse), records(path.join(root, "docs/repository-data"), path.join(root, "docs/chapter-2/quarantine.json"))]);
  const prompts = selectPrompts(evaluation);
  for (const entry of corpus.filter(item => !item.abstract?.trim()).sort((a, b) => a.uri.localeCompare(b.uri))) prompts.push({ id: `missing-${entry.uri}`, query: entry.title, expected: "insufficient" });
  for (const query of [
    "tesis tentang komputasi kuantum pada korpus ini",
    "tesis tentang paleontologi dinosaurus pada korpus ini",
    "tesis tentang budidaya tanaman mars pada korpus ini",
  ]) prompts.push({ id: `insufficient-${prompts.length}`, query, expected: "insufficient" });
  if (prompts.length !== 30) throw new Error(`expected 30 prompts, got ${prompts.length}`);
  const outputPath = path.resolve(output);
  let results = [];
  if (resume) results = JSON.parse(await readFile(outputPath, "utf8")).prompts ?? [];
  const completed = new Set(results.map(result => result.id));
  const persist = async () => writeFile(outputPath, `${JSON.stringify({ prompts: results, summary: summarize(results), review_required: true }, null, 2)}\n`);
  await run(baseURL, prompts.filter(prompt => !completed.has(prompt.id)), corpus, fetch, async result => {
    results.push(result);
    await persist();
    console.error(JSON.stringify({ completed: results.length, prompt: result.id, status: result.status }));
  });
  await persist();
  console.log(JSON.stringify(summarize(results)));
}
