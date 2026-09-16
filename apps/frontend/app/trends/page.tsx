"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

type Bucket = { value: string; count: number };
type Trends = {
  total: number;
  missing_abstracts: number;
  missing_divisions: number;
  missing_item_types: number;
  missing_subjects: number;
  by_year: Bucket[];
  by_division: Bucket[];
  by_item_type: Bucket[];
  by_subject: Bucket[];
};
type Filters = { years: number[]; divisions: string[] };

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "";

function endpoint(path: string, parameters?: Record<string, string>) {
  const base = apiURL.trim().replace(/\/+$/, "");
  if (!base) throw new Error("Frontend API URL is not configured.");
  const url = new URL(`${base}${path}`);
  for (const [key, value] of Object.entries(parameters ?? {})) if (value) url.searchParams.set(key, value);
  return url.toString();
}

function BucketList({ title, buckets }: { title: string; buckets: Bucket[] }) {
  return <section className="rounded-2xl border border-[#dadad3] bg-white p-5"><h2 className="text-lg font-semibold text-black">{title}</h2>{buckets.length === 0 ? <p className="mt-2 text-sm text-[#62625b]">No values in this filtered corpus.</p> : <ul className="mt-3 divide-y divide-[#e5e5e0]">{buckets.map((bucket) => <li key={bucket.value} className="flex items-baseline justify-between gap-4 py-2 text-sm"><span className="text-[#33332e]">{bucket.value}</span><span className="font-semibold text-black">{bucket.count.toLocaleString()}</span></li>)}</ul>}</section>;
}

export default function TrendsPage() {
  const [filters, setFilters] = useState<Filters>({ years: [], divisions: [] });
  const [selection, setSelection] = useState({ year: "", division: "" });
  const [trends, setTrends] = useState<Trends | null>(null);
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [error, setError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    void fetch(endpoint("/api/v1/filters"), { signal: controller.signal }).then(async (response) => {
      if (!response.ok) throw new Error("Filter options unavailable.");
      const body = await response.json() as Partial<Filters>;
      if (!Array.isArray(body.years) || !Array.isArray(body.divisions)) throw new Error("Invalid filter response.");
      setFilters({ years: body.years, divisions: body.divisions });
    }).catch((reason: unknown) => { if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : "Filter options unavailable."); });
    return () => controller.abort();
  }, []);

  useEffect(() => {
	const controller = new AbortController();
    void fetch(endpoint("/api/v1/trends", selection), { signal: controller.signal }).then(async (response) => {
      if (!response.ok) throw new Error("Corpus trends unavailable.");
      const body = await response.json() as Partial<Trends>;
      if (typeof body.total !== "number" || !Array.isArray(body.by_year) || !Array.isArray(body.by_division) || !Array.isArray(body.by_item_type) || !Array.isArray(body.by_subject)) throw new Error("Invalid trends response.");
      setTrends(body as Trends);
      setStatus("success");
    }).catch((reason: unknown) => { if (!controller.signal.aborted) { setError(reason instanceof Error ? reason.message : "Corpus trends unavailable."); setStatus("error"); } });
	return () => controller.abort();
	}, [selection]);

	function updateSelection(next: typeof selection) {
		setStatus("loading");
		setError("");
		setSelection(next);
	}

	return <div className="min-h-dvh bg-[#fbfbf9] text-[#33332e]">
		<header className="flex h-16 items-center justify-between border-b border-[#dadad3] bg-white px-4 sm:px-6 md:px-12">
			<Link href="/" className="text-lg font-bold tracking-tight text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">SearchLens</Link>
			<span className="rounded-full bg-[#f6f6f3] px-3 py-1.5 text-xs font-semibold text-[#62625b]">Corpus coverage</span>
		</header>
		<main className="mx-auto w-full max-w-5xl px-4 py-10 sm:px-6">
			<Link href="/" className="text-sm font-semibold text-[#33332e] underline decoration-[#91918c] underline-offset-4 focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">← Back to search</Link>
			<section className="mt-6"><h1 className="text-3xl font-bold tracking-tight text-black sm:text-4xl">Corpus trends</h1><p className="mt-2 max-w-2xl text-base leading-relaxed text-[#62625b]">Counts describe indexed repository coverage, not total institutional research output.</p></section>
			<div className="mt-6 grid gap-3 sm:grid-cols-2">
				<label className="text-xs font-semibold text-[#62625b]">Source year<select value={selection.year} onChange={(event) => updateSelection({ ...selection, year: event.target.value })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-white px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]"><option value="">All years</option>{filters.years.map((year) => <option key={year} value={year}>{year}</option>)}</select></label>
				<label className="text-xs font-semibold text-[#62625b]">Degree program<select value={selection.division} onChange={(event) => updateSelection({ ...selection, division: event.target.value })} className="mt-1 block min-h-11 w-full rounded-2xl border border-[#dadad3] bg-white px-3 text-sm text-black focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]"><option value="">All programs</option>{filters.divisions.map((division) => <option key={division} value={division}>{division}</option>)}</select></label>
			</div>
			{status === "loading" && <p className="mt-8 text-sm text-[#62625b]" role="status">Loading corpus trends…</p>}
			{status === "error" && <div className="mt-8 rounded-2xl border border-[#dadad3] bg-white p-6" role="alert"><p className="text-sm text-[#62625b]">{error}</p><button type="button" onClick={() => updateSelection({ ...selection })} className="mt-4 min-h-11 rounded-2xl bg-[#e5e5e0] px-4 text-sm font-bold text-black hover:bg-[#c8c8c1] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]">Try again</button></div>}
			{status === "success" && trends && <><section className="mt-8 rounded-2xl border border-[#dadad3] bg-white p-5"><p className="text-sm text-[#62625b]">Indexed records</p><p className="mt-1 text-3xl font-bold text-black">{trends.total.toLocaleString()}</p><dl className="mt-4 grid gap-3 text-sm sm:grid-cols-4"><div><dt className="text-[#62625b]">Missing abstracts</dt><dd className="font-semibold text-black">{trends.missing_abstracts.toLocaleString()}</dd></div><div><dt className="text-[#62625b]">Missing programs</dt><dd className="font-semibold text-black">{trends.missing_divisions.toLocaleString()}</dd></div><div><dt className="text-[#62625b]">Missing item types</dt><dd className="font-semibold text-black">{trends.missing_item_types.toLocaleString()}</dd></div><div><dt className="text-[#62625b]">Missing subjects</dt><dd className="font-semibold text-black">{trends.missing_subjects.toLocaleString()}</dd></div></dl></section><div className="mt-5 grid gap-5 md:grid-cols-2"><BucketList title="By source year" buckets={trends.by_year} /><BucketList title="By degree program" buckets={trends.by_division} /><BucketList title="By item type" buckets={trends.by_item_type} /><BucketList title="By normalized subject" buckets={trends.by_subject} /></div></>}
		</main>
	</div>;
}
