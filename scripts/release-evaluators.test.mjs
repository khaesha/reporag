import assert from "node:assert/strict";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { selectPrompts } from "./evaluate-synthesis.mjs";
import { records, verify } from "./verify-exact-titles.mjs";

test("exact-title records exclude quarantine and duplicate URIs", async () => {
  const directory = await mkdtemp(path.join(os.tmpdir(), "searchlens-release-"));
  try {
    await writeFile(path.join(directory, "repository_2025_data.json"), JSON.stringify([
      { title: "One", uri: "keep" }, { title: "Duplicate", uri: "keep" }, { title: "Skip", uri: "skip" }, { title: "", uri: "empty" },
    ]));
    const quarantine = path.join(directory, "quarantine.json");
    await writeFile(quarantine, JSON.stringify([{ uri: "skip" }]));
    assert.deepEqual(await records(directory, quarantine), [{ title: "One", uri: "keep" }]);
  } finally {
    await rm(directory, { recursive: true });
  }
});

test("exact-title verification accepts an unambiguous result within the result page", async () => {
  const originalFetch = global.fetch;
  global.fetch = async () => new Response(JSON.stringify({ results: [{ uri: "other" }, { uri: "wanted" }] }));
  try {
    assert.deepEqual(await verify("http://example.test", [{ title: "Same", uri: "wanted" }]), { checked: 1, failures: [], passed: 1 });
  } finally {
    global.fetch = originalFetch;
  }
});

test("synthesis sample evenly selects the three judged intents", () => {
  const evaluation = ["conceptual", "bilingual", "abbreviation"].flatMap(intent => ["A", "B", "C"].flatMap(division => Array.from({ length: 8 }, (_, index) => ({ id: `${intent}-${division}-${index}`, intent, division, query: `${intent} ${division} ${index}` }))));
  const prompts = selectPrompts(evaluation);
  assert.equal(prompts.length, 24);
  for (const intent of ["conceptual", "bilingual", "abbreviation"]) assert.equal(prompts.filter(prompt => prompt.id.startsWith(intent)).length, 8);
});
