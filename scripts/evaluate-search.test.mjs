import assert from "node:assert/strict";
import test from "node:test";

import { aggregate, scoreQuery, validateEvaluation } from "./evaluate-search.mjs";

test("scores multiple graded judgments", () => {
  const judgments = [
    { uri: "best", relevance: 3 },
    { uri: "related", relevance: 2 },
    { uri: "partial", relevance: 1 },
  ];
  const score = scoreQuery(judgments, ["partial", "best", "related"]);
  assert.equal(score.recall_at_10, 1);
  assert.equal(score.mrr_at_10, 0.5);
  assert.equal(score.top_5_success, 1);
  assert.ok(score.ndcg_at_10 > 0 && score.ndcg_at_10 < 1);
  assert.deepEqual(aggregate([score, score]), {
    recall_at_10: 1,
    mrr_at_10: 0.5,
    ndcg_at_10: Number(score.ndcg_at_10.toFixed(4)),
    top_5_success: 1,
  });
});

test("validates evaluation boundaries", () => {
  assert.throws(() => validateEvaluation([]), /at least 100/);
  const items = Array.from({ length: 100 }, (_, index) => ({
    id: `q-${index}`,
    query: `query ${index}`,
    division: "CS",
    intent: "conceptual",
    judgments: [{ uri: `uri-${index}`, relevance: 3 }],
  }));
  assert.doesNotThrow(() => validateEvaluation(items));
  items[1].id = items[0].id;
  assert.throws(() => validateEvaluation(items), /duplicate id/);
});
