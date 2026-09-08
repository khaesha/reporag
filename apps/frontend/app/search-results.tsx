export type SearchResult = {
  title: string;
  abstract: string | null;
  authors: string[];
  item_type: string | null;
  subjects: string | null;
  divisions: string | null;
  date_deposited: string | null;
  source_year: number;
  uri: string;
  score: number;
};

export type SearchResponse = {
  query: string;
  page: number;
  limit: number;
  total: number;
  results: SearchResult[];
};

export type FilterValues = {
  years: number[];
  divisions: string[];
  item_types: string[];
};

export type SearchSelection = {
  year: string;
  division: string;
  itemType: string;
  hasAbstract: "" | "true" | "false";
  sort: "relevance" | "title" | "date";
};

type Props = {
  response: SearchResponse | null;
  status: "idle" | "loading" | "success" | "error";
  error: string;
  selection: SearchSelection;
  filterValues: FilterValues;
  filterStatus: "loading" | "success" | "error";
  filterError: string;
  onSelectionChange: (selection: SearchSelection) => void;
  onPageChange: (page: number) => void;
  onRetry: () => void;
  onResetFilters: () => void;
  onRetryFilters: () => void;
};

const sortOptions = [
  ["relevance", "Relevance"],
  ["title", "Title A–Z"],
  ["date", "Newest deposit"],
] as const;

export default function SearchResults({
  response,
  status,
  error,
  selection,
  filterValues,
  filterStatus,
  filterError,
  onSelectionChange,
  onPageChange,
  onRetry,
  onResetFilters,
  onRetryFilters,
}: Props) {
  const filtersDisabled = filterStatus !== "success";
  const totalPages = response
    ? Math.max(1, Math.ceil(response.total / response.limit))
    : 1;
  const hasFilters = Boolean(
    selection.year ||
      selection.division ||
      selection.itemType ||
      selection.hasAbstract,
  );

  return (
    <section className="mt-6 w-full max-w-3xl text-left" aria-label="Search results" aria-busy={status === "loading"}>
      <div className="mb-4 space-y-4 border-b border-[#dadad3] py-4">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <label className="text-xs font-semibold text-[#62625b]">
            Year
            <select value={selection.year} disabled={filtersDisabled} onChange={(event) => onSelectionChange({ ...selection, year: event.target.value })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-[#f6f6f3] px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:text-[#91918c]">
              <option value="">All years</option>
              {filterValues.years.map((year) => <option key={year} value={year}>{year}</option>)}
            </select>
          </label>
          <label className="text-xs font-semibold text-[#62625b]">
            Degree program
            <select value={selection.division} disabled={filtersDisabled} onChange={(event) => onSelectionChange({ ...selection, division: event.target.value })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-[#f6f6f3] px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:text-[#91918c]">
              <option value="">All programs</option>
              {filterValues.divisions.map((division) => <option key={division} value={division}>{division}</option>)}
            </select>
          </label>
          <label className="text-xs font-semibold text-[#62625b]">
            Item type
            <select value={selection.itemType} disabled={filtersDisabled} onChange={(event) => onSelectionChange({ ...selection, itemType: event.target.value })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-[#f6f6f3] px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:text-[#91918c]">
              <option value="">All item types</option>
              {filterValues.item_types.map((itemType) => <option key={itemType} value={itemType}>{itemType}</option>)}
            </select>
          </label>
          <label className="text-xs font-semibold text-[#62625b]">
            Abstract
            <select value={selection.hasAbstract} onChange={(event) => onSelectionChange({ ...selection, hasAbstract: event.target.value as "" | "true" | "false" })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-[#f6f6f3] px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">
              <option value="">Any availability</option>
              <option value="true">Available</option>
              <option value="false">Unavailable</option>
            </select>
          </label>
        </div>

        {filterStatus === "error" && (
          <p className="text-sm text-[#62625b]" role="status">
            {filterError}{" "}
            <button type="button" onClick={onRetryFilters} className="min-h-11 rounded-2xl px-2 font-semibold text-black underline decoration-[#91918c] underline-offset-4 focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">Retry filters</button>
          </p>
        )}

        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <span className="text-sm font-bold text-[#62625b]">Sort by</span>
          <div className="flex flex-wrap gap-2" aria-label="Sort results">
            {sortOptions.map(([value, label]) => (
              <button key={value} type="button" aria-pressed={selection.sort === value} onClick={() => onSelectionChange({ ...selection, sort: value })} className={`min-h-11 rounded-full px-4 text-sm font-bold focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] ${selection.sort === value ? "bg-black text-white" : "bg-[#f6f6f3] text-black hover:bg-[#dadad3]"}`}>
                {label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {status === "loading" && (
        <div className="rounded-[32px] border border-[#dadad3] bg-white p-8 text-center" role="status" aria-live="polite">
          <div className="mx-auto mb-4 size-10 rounded-full border-4 border-[#dadad3] border-t-black motion-safe:animate-spin" />
          <h2 className="text-xl font-semibold text-black">Searching thesis metadata…</h2>
          <p className="mt-2 text-sm text-[#62625b]">Checking titles, authors, subjects, degree programs, and available abstracts.</p>
        </div>
      )}

      {status === "error" && (
        <div className="rounded-[32px] border border-[#dadad3] bg-white p-8 text-center" role="alert">
          <h2 className="text-xl font-semibold text-black">Search failed</h2>
          <p className="mt-2 text-sm text-[#62625b]">{error}</p>
          <button type="button" onClick={onRetry} className="mt-5 min-h-11 rounded-2xl bg-[#e5e5e0] px-5 text-sm font-bold text-black hover:bg-[#c8c8c1] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">Try again</button>
        </div>
      )}

      {status === "success" && response?.total === 0 && (
        <div className="rounded-[32px] border border-[#dadad3] bg-white p-8 text-center" aria-live="polite">
          <h2 className="text-xl font-semibold text-black">No theses found</h2>
          <p className="mt-2 text-sm text-[#62625b]">Try different keywords or remove some filters.</p>
          {hasFilters && (
            <button type="button" onClick={onResetFilters} className="mt-5 min-h-11 rounded-2xl bg-[#e5e5e0] px-5 text-sm font-bold text-black hover:bg-[#c8c8c1] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">Reset filters</button>
          )}
        </div>
      )}

      {status === "success" && response && response.total > 0 && (
        <>
          <p className="mb-4 text-sm font-medium text-[#62625b]" aria-live="polite">
            {response.total.toLocaleString()} result{response.total === 1 ? "" : "s"} for “{response.query}”
          </p>
          <ol className="space-y-3">
            {response.results.map((result) => (
              <li key={result.uri}>
                <article className="rounded-2xl border border-[#dadad3] bg-white p-5 hover:border-[#91918c]">
                  <h2 className="text-lg leading-snug font-semibold text-black">
                    <a href={result.uri} className="rounded-sm hover:underline focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">{result.title}</a>
                  </h2>
                  <p className="mt-1 text-sm text-[#62625b]">{result.authors.length > 0 ? result.authors.join(", ") : "Author unavailable"}</p>
                  <div className="mt-3 flex flex-wrap gap-2 text-xs font-semibold text-[#62625b]">
                    <span className="rounded-full bg-[#f6f6f3] px-3 py-1.5">{result.source_year}</span>
                    <span className="rounded-full bg-[#f6f6f3] px-3 py-1.5">{result.item_type ?? "Item type unavailable"}</span>
                    <span className="rounded-full bg-[#f6f6f3] px-3 py-1.5">{result.divisions ?? "Degree program unavailable"}</span>
                  </div>
                  {result.abstract ? (
                    <p className="mt-3 line-clamp-4 text-base leading-[1.5] text-[#33332e]">{result.abstract}</p>
                  ) : (
                    <p className="mt-3 text-sm font-semibold text-[#62625b]">Abstract unavailable</p>
                  )}
                  <a href={result.uri} className="mt-4 inline-flex min-h-11 items-center rounded-2xl px-1 text-sm font-bold text-[#33332e] underline decoration-[#91918c] underline-offset-4 focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">
                    Open repository record <span aria-hidden="true" className="ml-1">→</span>
                  </a>
                </article>
              </li>
            ))}
          </ol>

          <nav aria-label="Search result pages" className="mt-8 mb-12 flex items-center justify-center gap-3">
            <button type="button" disabled={response.page <= 1} onClick={() => onPageChange(response.page - 1)} className="min-h-11 rounded-2xl bg-[#e5e5e0] px-4 text-sm font-bold text-black hover:bg-[#c8c8c1] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:opacity-50">Previous</button>
            <span className="text-sm font-semibold text-[#62625b]" aria-current="page">Page {response.page} of {totalPages}</span>
            <button type="button" disabled={response.page >= totalPages} onClick={() => onPageChange(response.page + 1)} className="min-h-11 rounded-2xl bg-[#e5e5e0] px-4 text-sm font-bold text-black hover:bg-[#c8c8c1] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:opacity-50">Next</button>
          </nav>
        </>
      )}
    </section>
  );
}
