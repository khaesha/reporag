"use client";

import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import SearchResults, {
  FilterValues,
  SearchResponse,
  SearchSelection,
} from "./search-results";
import { buildSearchURL } from "./search-url.mjs";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "";
const emptyFilters: FilterValues = {
  years: [],
  divisions: [],
  item_types: [],
};
const defaultSelection: SearchSelection = {
  year: "",
  division: "",
  itemType: "",
  hasAbstract: "",
  sort: "relevance",
};
const suggestions = [
  ["Robot line follower", "robot pengikut garis mikrokontroler"],
  ["Social commerce", "jual beli online social commerce"],
  ["Data analysis", "analisis data descriptive statistic web"],
  ["Language models", "non-playable character large language model"],
] as const;

type RequestStatus = "idle" | "loading" | "success" | "error";
type FilterStatus = "loading" | "success" | "error";

async function errorMessage(response: Response, fallback: string) {
  try {
    const body = (await response.json()) as {
      error?: { message?: unknown };
    };
    return typeof body.error?.message === "string"
      ? body.error.message
      : fallback;
  } catch {
    return fallback;
  }
}

function isSearchResponse(value: unknown): value is SearchResponse {
  if (!value || typeof value !== "object") return false;
  const response = value as Partial<SearchResponse>;
  return (
    typeof response.query === "string" &&
    typeof response.page === "number" &&
    typeof response.limit === "number" &&
    typeof response.total === "number" &&
    Array.isArray(response.results)
  );
}

function isFilterValues(value: unknown): value is FilterValues {
  if (!value || typeof value !== "object") return false;
  const filters = value as Partial<FilterValues>;
  return (
    Array.isArray(filters.years) &&
    Array.isArray(filters.divisions) &&
    Array.isArray(filters.item_types)
  );
}

async function fetchFilterValues(signal: AbortSignal) {
  if (!apiURL.trim()) throw new Error("Frontend API URL is not configured.");
  const base = apiURL.trim().replace(/\/+$/, "");
  const result = await fetch(`${base}/api/v1/filters`, { signal });
  if (!result.ok) {
    throw new Error(await errorMessage(result, "Filter options unavailable."));
  }
  const body: unknown = await result.json();
  if (!isFilterValues(body)) throw new Error("Invalid filter response.");
  return body;
}

export default function SearchForm() {
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [selection, setSelection] =
    useState<SearchSelection>(defaultSelection);
  const [status, setStatus] = useState<RequestStatus>("idle");
  const [response, setResponse] = useState<SearchResponse | null>(null);
  const [error, setError] = useState("");
  const [filterValues, setFilterValues] =
    useState<FilterValues>(emptyFilters);
  const [filterStatus, setFilterStatus] =
    useState<FilterStatus>("loading");
  const [filterError, setFilterError] = useState("");
  const searchRequest = useRef<AbortController>(null);
  const filterRequest = useRef<AbortController>(null);
  const input = useRef<HTMLInputElement>(null);
  const lastPage = useRef(1);
  const disabled = !query.trim();

  const loadFilters = useCallback(async () => {
    filterRequest.current?.abort();
    const controller = new AbortController();
    filterRequest.current = controller;
    setFilterStatus("loading");
    setFilterError("");

    try {
      const body = await fetchFilterValues(controller.signal);
      if (filterRequest.current !== controller) return;
      setFilterValues(body);
      setFilterStatus("success");
    } catch (requestError) {
      if (controller.signal.aborted) return;
      setFilterStatus("error");
      setFilterError(
        requestError instanceof Error
          ? requestError.message
          : "Filter options unavailable.",
      );
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    filterRequest.current = controller;
    void fetchFilterValues(controller.signal)
      .then((body) => {
        if (filterRequest.current !== controller) return;
        setFilterValues(body);
        setFilterStatus("success");
      })
      .catch((requestError: unknown) => {
        if (controller.signal.aborted) return;
        setFilterStatus("error");
        setFilterError(
          requestError instanceof Error
            ? requestError.message
            : "Filter options unavailable.",
        );
      });
    return () => {
      searchRequest.current?.abort();
      filterRequest.current?.abort();
    };
  }, []);

  const runSearch = useCallback(
    async (nextQuery: string, nextSelection: SearchSelection, page: number) => {
      searchRequest.current?.abort();
      const controller = new AbortController();
      searchRequest.current = controller;
      lastPage.current = page;
      setStatus("loading");
      setResponse(null);
      setError("");

      try {
        const result = await fetch(
          buildSearchURL(apiURL, {
            query: nextQuery,
            page,
            limit: 10,
            sort: nextSelection.sort,
            year: nextSelection.year,
            division: nextSelection.division,
            itemType: nextSelection.itemType,
            hasAbstract: nextSelection.hasAbstract,
          }),
          { signal: controller.signal },
        );
        if (!result.ok) {
          throw new Error(await errorMessage(result, "Search unavailable."));
        }
        const body: unknown = await result.json();
        if (!isSearchResponse(body)) throw new Error("Invalid search response.");
        if (searchRequest.current !== controller) return;
        setResponse(body);
        setStatus("success");
      } catch (requestError) {
        if (controller.signal.aborted) return;
        setStatus("error");
        setError(
          requestError instanceof Error
            ? requestError.message
            : "Search service unavailable. Try again.",
        );
      }
    },
    [],
  );

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextQuery = query.trim();
    if (!nextQuery) return;
    setSubmittedQuery(nextQuery);
    void runSearch(nextQuery, selection, 1);
  }

  function changeSelection(nextSelection: SearchSelection) {
    setSelection(nextSelection);
    if (submittedQuery) void runSearch(submittedQuery, nextSelection, 1);
  }

  function changePage(page: number) {
    void runSearch(submittedQuery, selection, page);
  }

  function retrySearch() {
    void runSearch(submittedQuery, selection, lastPage.current);
  }

  function resetFilters() {
    setSelection(defaultSelection);
    void runSearch(submittedQuery, defaultSelection, 1);
  }

  return (
    <>
      <form className="mb-6 w-full max-w-3xl" onSubmit={submit}>
        <label htmlFor="thesis-query" className="sr-only">
          Search thesis metadata and available abstracts
        </label>
        <div className="flex h-12 items-center rounded-full border border-[#dadad3] bg-[#f6f6f3] pr-1.5 pl-4 focus-within:border-[#91918c] focus-within:bg-white focus-within:shadow-[0_0_0_2px_#000,0_0_0_6px_#435ee5]">
          <svg aria-hidden="true" className="mr-2 size-5 shrink-0 text-[#91918c]" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2.2" strokeLinecap="round">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            ref={input}
            id="thesis-query"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search by title, author, or topic"
            autoComplete="off"
            maxLength={200}
            className="h-full min-w-0 flex-1 bg-transparent text-base text-black outline-none placeholder:text-[#91918c]"
          />
          <button
            type="submit"
            disabled={disabled}
            aria-label="Search"
            className="flex size-10 shrink-0 items-center justify-center rounded-full bg-[#e60023] text-white enabled:hover:bg-[#cc001f] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:bg-[#f6f6f3] disabled:text-[#91918c]"
          >
            <svg aria-hidden="true" className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14m-7-7 7 7-7 7" />
            </svg>
          </button>
        </div>
        <div className="mt-2.5 flex flex-col gap-2 px-3 text-xs text-[#62625b] sm:flex-row sm:items-center sm:justify-between">
          <span>
            Press <kbd className="rounded border border-[#dadad3] bg-[#f6f6f3] px-1.5 py-0.5 font-mono text-[11px] text-black">Enter</kbd>{" "}
            to search
          </span>
          <span>Repository metadata and available abstracts</span>
        </div>
      </form>

      {!submittedQuery && (
        <div className="flex w-full max-w-3xl flex-wrap items-center justify-center gap-2 pt-2">
          <span className="mr-1 text-xs font-semibold tracking-wider text-[#91918c] uppercase">Try searching:</span>
          {suggestions.map(([label, value]) => (
            <button key={label} type="button" onClick={() => { setQuery(value); input.current?.focus(); }} className="min-h-11 rounded-full bg-[#f6f6f3] px-3.5 text-xs font-semibold text-black hover:bg-[#dadad3] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">
              {label}
            </button>
          ))}
        </div>
      )}

      {submittedQuery && (
        <SearchResults
          response={response}
          status={status}
          error={error}
          selection={selection}
          filterValues={filterValues}
          filterStatus={filterStatus}
          filterError={filterError}
          onSelectionChange={changeSelection}
          onPageChange={changePage}
          onRetry={retrySearch}
          onResetFilters={resetFilters}
          onRetryFilters={() => void loadFilters()}
        />
      )}
    </>
  );
}
