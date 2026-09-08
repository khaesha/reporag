import assert from "node:assert/strict";
import test from "node:test";

import { buildSearchURL } from "./search-url.mjs";

test("builds encoded search URLs and omits empty filters", () => {
  const url = new URL(
    buildSearchURL("http://localhost:8080/", {
      query: "robot & control",
      page: 2,
      limit: 10,
      sort: "date",
      year: "2024",
      division: "Computer Science",
      itemType: "",
      hasAbstract: "false",
    }),
  );

  assert.equal(url.pathname, "/api/v1/search");
  assert.deepEqual(Object.fromEntries(url.searchParams), {
    q: "robot & control",
    page: "2",
    limit: "10",
    sort: "date",
    year: "2024",
    division: "Computer Science",
    has_abstract: "false",
  });
});

test("requires an API URL", () => {
  assert.throws(
    () =>
      buildSearchURL(" ", {
        query: "robot",
        page: 1,
        limit: 10,
        sort: "relevance",
      }),
    /NEXT_PUBLIC_API_URL/,
  );
});
