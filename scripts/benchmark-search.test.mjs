import assert from "node:assert/strict";
import test from "node:test";

import { percentile, serverTiming } from "./benchmark-search.mjs";

test("percentile uses nearest-rank values without mutating samples", () => {
  const samples = [30, 10, 20, 40];
  assert.equal(percentile(samples, 0.5), 20);
  assert.equal(percentile(samples, 0.95), 40);
  assert.deepEqual(samples, [30, 10, 20, 40]);
  assert.throws(() => percentile([], 0.95), /requires finite samples/);
});

test("reads named Server-Timing durations", () => {
  const header = "model;dur=125.25, retrieval;dur=18.50";
  assert.equal(serverTiming(header, "model"), 125.25);
  assert.equal(serverTiming(header, "retrieval"), 18.5);
  assert.throws(() => serverTiming(header, "generation"), /missing generation/);
});
