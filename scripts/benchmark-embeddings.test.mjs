import assert from "node:assert/strict";
import test from "node:test";

import { canonicalInput, rankByCosine, selectWinner, validateEmbeddingResponse } from "./benchmark-embeddings.mjs";

test("builds canonical input with empty optional fields", () => {
  assert.equal(
    canonicalInput({ title: "One", authors: ["A", "B"], divisions: "CS" }),
    "Title: One\nAuthors: A, B\nSubjects: \nDivision: CS\nAbstract: ",
  );
});

test("ranks cosine similarity with URI tie break", () => {
  assert.deepEqual(
    rankByCosine([1, 0], [
      { uri: "b", embedding: [1, 0] },
      { uri: "a", embedding: [2, 0] },
      { uri: "c", embedding: [0, 1] },
    ]).map(({ uri }) => uri),
    ["a", "b", "c"],
  );
});

test("validates provider response dimensions without mutating it", () => {
  const body = { data: [{ index: 1, embedding: [0, 1] }, { index: 0, embedding: [1, 0] }] };
  assert.deepEqual(validateEmbeddingResponse(body, 2, 2, "model"), [[1, 0], [0, 1]]);
  assert.equal(body.data[0].index, 1);
  assert.throws(() => validateEmbeddingResponse(body, 2, 3, "model"), /dimensions/);
});

test("selects quality unless models are within tie threshold", () => {
  const quality = selectWinner([
    { model: "a", dimensions: 2, subset_ndcg_at_10: 0.7, estimated_cost_usd: 2, p95_ms: 1 },
    { model: "b", dimensions: 2, subset_ndcg_at_10: 0.68, estimated_cost_usd: 1, p95_ms: 1 },
  ], 0.6);
  assert.equal(quality.model, "a");
  assert.equal(quality.passed, true);
  const cost = selectWinner([
    { model: "a", dimensions: 2, subset_ndcg_at_10: 0.7, estimated_cost_usd: 2, p95_ms: 1 },
    { model: "b", dimensions: 2, subset_ndcg_at_10: 0.695, estimated_cost_usd: 1, p95_ms: 2 },
  ], 0.71);
  assert.equal(cost.model, "b");
  assert.equal(cost.passed, false);
});
