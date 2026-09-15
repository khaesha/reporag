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
    assert.match(result.corpus_sha256, /^[a-f0-9]{64}$/);
    assert.equal(result.valid_unique_uris, 1);
    assert.equal(result.invalid_field_count, 0);
    assert.deepEqual(result.degree_programs, [{ value: "CS", count: 2 }]);
    assert.equal(result.missing_fields.abstract, 2);
    assert.equal(result.missing_fields.authors, 1);
    assert.deepEqual(result.duplicate_uris, [{ uri: "u1", count: 2 }]);
  } finally {
    await rm(directory, { recursive: true });
  }
});

test("reports invalid values, probable duplicate titles, and quarantine", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "searchlens-audit-quality-"));
  try {
    const file = path.join(directory, "repository_2025_data.json");
    await writeFile(file, JSON.stringify([
      { title: "Same title!", authors: [], divisions: "CS", date_deposited: "31 Feb 2025 10:00", uri: "bad" },
      { title: "same title", authors: [], divisions: "CS", uri: "keep" },
    ]));
    const result = await auditCorpus(directory, {
      normalization: { divisions: { CS: "CS" }, item_types: {}, subject_classes: {} },
      quarantine: [{ uri: "bad", reason: "fixture" }],
    });
    assert.equal(result.valid_unique_uris, 1);
    assert.equal(result.quarantined_uri_count, 1);
    assert.equal(result.invalid_field_count, 1);
    assert.equal(result.invalid_fields[0].field, "date_deposited");
    assert.equal(result.probable_duplicate_title_count, 0, "quarantined titles do not become duplicate candidates");
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

test("chapter 2 evaluation covers every program and intent", async () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const evaluation = JSON.parse(await readFile(path.join(root, "docs/chapter-2/evaluation.json"), "utf8"));
  const normalization = JSON.parse(await readFile(path.join(root, "docs/chapter-2/normalization.json"), "utf8"));
  const quarantine = new Set(
    JSON.parse(await readFile(path.join(root, "docs/chapter-2/quarantine.json"), "utf8")).map(({ uri }) => uri),
  );
  const files = await findCorpusFiles(path.join(root, "docs/repository-data"));
  const records = (await Promise.all(files.map(async (file) => JSON.parse(await readFile(file, "utf8"))))).flat();
  const byURI = new Map(records.map((record) => [record.uri, record]));
  const intents = new Set(evaluation.map(({ intent }) => intent));
  const divisions = new Set(evaluation.map(({ division }) => division));

  assert.equal(evaluation.length, 100);
  assert.deepEqual(intents, new Set(["exact_title", "exact_author", "lexical", "conceptual", "bilingual", "abbreviation"]));
  assert.deepEqual(divisions, new Set(Object.values(normalization.divisions)));
  for (const item of evaluation) {
    for (const { uri } of item.judgments) {
      assert.ok(byURI.has(uri), `${item.id}: ${uri} is not in corpus`);
      assert.ok(!quarantine.has(uri), `${item.id}: ${uri} is quarantined`);
      assert.equal(normalization.divisions[byURI.get(uri).divisions], item.division, `${item.id}: division mismatch`);
    }
  }
});
