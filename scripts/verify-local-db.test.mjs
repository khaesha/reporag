import assert from "node:assert/strict";
import test from "node:test";

import { disposableDatabaseURL, disposableNames } from "./verify-local-db.mjs";

test("disposable database URLs retain local connection details", () => {
  assert.equal(disposableDatabaseURL("postgres://user:pass@127.0.0.1:55432/searchlens?sslmode=disable", "searchlens_phase3_build"), "postgres://user:pass@127.0.0.1:55432/searchlens_phase3_build?sslmode=disable");
});

test("disposable database URLs reject non-loopback targets", () => {
  assert.throws(() => disposableDatabaseURL("postgres://user:pass@example.test/searchlens", "restore"), /loopback/);
});

test("disposable names are distinct and identifier-safe", () => {
  const names = disposableNames(1, "abc123");
  assert.deepEqual(names, { build: "searchlens_phase3_build_1_abc123", restore: "searchlens_phase3_restore_1_abc123" });
});
