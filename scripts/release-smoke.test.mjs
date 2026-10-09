import assert from "node:assert/strict";
import test from "node:test";

import { smoke } from "./release-smoke.mjs";

test("release smoke validates every required API surface", async () => {
  const fetchFn = async (url) => {
    const pathname = new URL(url).pathname;
    if (pathname === "/readyz") return new Response(JSON.stringify({ status: "ready" }));
    if (pathname === "/api/v1/search") return new Response(JSON.stringify({ results: [{ uri: "https://example.test/1" }] }));
    if (pathname === "/api/v1/related") return new Response(JSON.stringify({ results: [{ uri: "https://example.test/2" }] }));
    if (pathname === "/api/v1/trends") return new Response(JSON.stringify({ total: 1 }));
    if (pathname === "/api/v1/answer") return new Response(JSON.stringify({ insufficient_evidence: false, citations: [{ id: 1 }] }));
    throw new Error(`unexpected ${pathname}`);
  };
  const result = await smoke("http://example.test", fetchFn);
  assert.deepEqual(result.checks, ["ready", "lexical", "semantic", "hybrid", "related", "trends", "synthesis"]);
});
