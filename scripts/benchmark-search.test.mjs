import assert from "node:assert/strict";
import test from "node:test";

import { percentile } from "./benchmark-search.mjs";

test("percentile uses nearest-rank values without mutating samples", () => {
  const samples = [30, 10, 20, 40];
  assert.equal(percentile(samples, 0.5), 20);
  assert.equal(percentile(samples, 0.95), 40);
  assert.deepEqual(samples, [30, 10, 20, 40]);
  assert.throws(() => percentile([], 0.95), /requires finite samples/);
});
