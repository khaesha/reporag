import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { auditCorpus, findCorpusFiles } from "./audit-corpus.mjs";

test("audits nested corpus files", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "searchlens-audit-"));
  try {
    await mkdir(path.join(directory, "nested"));
    await writeFile(
      path.join(directory, "repository_2024_data.json"),
      JSON.stringify([{ title: "One", abstract: null, authors: [], divisions: "CS", uri: "u1" }]),
    );
    await writeFile(
      path.join(directory, "nested/repository_2025_data.json"),
      JSON.stringify([{ title: "Two", authors: ["A"], divisions: "CS", uri: "u1" }]),
    );
    await writeFile(path.join(directory, "ignored.json"), "not json");

    const result = await auditCorpus(directory);
    assert.equal(result.files, 2);
    assert.equal(result.records, 2);
    assert.equal(result.unique_uris, 1);
    assert.equal(result.duplicate_uri_count, 1);
    assert.deepEqual(result.degree_programs, [{ value: "CS", count: 2 }]);
    assert.equal(result.missing_fields.abstract, 2);
    assert.equal(result.missing_fields.authors, 1);
    assert.deepEqual(result.duplicate_uris, [{ uri: "u1", count: 2 }]);
  } finally {
    await rm(directory, { recursive: true });
  }
});

test("evaluation set references corpus records", async () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const evaluation = JSON.parse(await readFile(path.join(root, "docs/chapter-1/evaluation.json"), "utf8"));
  const files = await findCorpusFiles(path.join(root, "docs/repository-data"));
  const records = (await Promise.all(files.map(async (file) => JSON.parse(await readFile(file, "utf8"))))).flat();
  const uris = new Set(records.map((record) => record.uri));

  assert.ok(evaluation.length >= 30);
  assert.equal(new Set(evaluation.map(({ query }) => query)).size, evaluation.length);
  for (const item of evaluation) {
    assert.ok(["exact_title", "exact_author", "topic"].includes(item.kind));
    assert.equal(item.max_rank, item.kind === "topic" ? 5 : 1);
    assert.ok(item.expected_uris.length > 0);
    for (const uri of item.expected_uris) assert.ok(uris.has(uri), `${uri} is not in corpus`);
  }
});
