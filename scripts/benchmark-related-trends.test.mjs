import assert from "node:assert/strict";
import test from "node:test";

import { benchmark } from "./benchmark-related-trends.mjs";

test("benchmarks related and trends retrieval timing separately", async () => {
  const calls = [];
  const fetchFn = async (url) => {
    calls.push(new URL(url).pathname);
    const related = new URL(url).pathname.endsWith("/related");
    return new Response(JSON.stringify(related ? { results: [] } : { total: 2190 }), {
      headers: { "server-timing": "retrieval;dur=1.25" },
    });
  };
  const result = await benchmark("http://example.test", "https://example.test/1", 2, fetchFn);
  assert.equal(result.related.samples, 2);
  assert.equal(result.related.retrieval_p95_ms, 1.25);
  assert.equal(result.trends.samples, 2);
  assert.equal(result.trends.retrieval_p95_ms, 1.25);
  assert.deepEqual(calls, ["/api/v1/related", "/api/v1/related", "/api/v1/related", "/api/v1/trends", "/api/v1/trends", "/api/v1/trends"]);
});
