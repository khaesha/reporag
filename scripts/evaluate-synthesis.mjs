import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
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

export async function run(baseURL, prompts, corpus) {
  const byURI = new Map(corpus.map(item => [item.uri, item]));
  const results = [];
  for (const prompt of prompts) {
    const response = await fetch(new URL("/api/v1/answer", baseURL), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ query: prompt.query }), signal: AbortSignal.timeout(75_000) });
    const body = await response.json();
    const citations = (body.citations ?? []).map(citation => ({ ...citation, corpus_match: byURI.get(citation.uri)?.title === citation.title, abstract_available: Boolean(byURI.get(citation.uri)?.abstract?.trim()) }));
    results.push({ ...prompt, request_id: response.headers.get("x-request-id"), status: response.status, answer: body.answer ?? "", insufficient_evidence: body.insufficient_evidence === true, citations, citation_valid: citations.every(item => item.corpus_match), claims: [] });
  }
  return results;
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  const root = path.resolve(path.dirname(script), "..");
  const [baseURL = "http://localhost:8080", output = path.join(root, "docs/chapter-2/synthesis-results.json")] = process.argv.slice(2);
  const [evaluation, corpus] = await Promise.all([readFile(path.join(root, "docs/chapter-2/evaluation.json"), "utf8").then(JSON.parse), records(path.join(root, "docs/repository-data"), path.join(root, "docs/chapter-2/quarantine.json"))]);
  const prompts = selectPrompts(evaluation);
  for (const entry of corpus.filter(item => !item.abstract?.trim()).sort((a, b) => a.uri.localeCompare(b.uri))) prompts.push({ id: `missing-${entry.uri}`, query: entry.title, expected: "insufficient" });
  for (const query of [
    "tesis tentang komputasi kuantum pada korpus ini",
    "tesis tentang paleontologi dinosaurus pada korpus ini",
    "tesis tentang budidaya tanaman mars pada korpus ini",
  ]) prompts.push({ id: `insufficient-${prompts.length}`, query, expected: "insufficient" });
  if (prompts.length !== 30) throw new Error(`expected 30 prompts, got ${prompts.length}`);
  const results = await run(baseURL, prompts, corpus);
  await writeFile(path.resolve(output), `${JSON.stringify({ prompts: results, review_required: true }, null, 2)}\n`);
  console.log(JSON.stringify({ prompts: results.length, citations_valid: results.every(item => item.citation_valid) }));
}
