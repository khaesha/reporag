"use client";

import Image from "next/image";
import { FormEvent, ReactNode, useRef, useState } from "react";

const suggestions = [
  ["Summarize design tokens", "Summarize latest design tokens update"],
  ["Extract Q3 metrics", "Extract key metrics from Q3 report"],
  ["Vector search vs keyword", "Compare vector search vs keyword search"],
  ["API auth snippet", "Generate code snippet for API auth"],
] as const;

export default function SearchForm({ results }: { results: ReactNode }) {
  const [query, setQuery] = useState("");
  const [state, setState] = useState<"idle" | "loading" | "results">("idle");
  const inputRef = useRef<HTMLInputElement>(null);
  const timerRef = useRef<number>(undefined);
  const disabled = !query.trim();

  function chooseSuggestion(value: string) {
    setQuery(value);
    inputRef.current?.focus();
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (disabled) return;
    setState("loading");
    timerRef.current = window.setTimeout(() => setState("results"), 1800);
  }

  function cancel() {
    window.clearTimeout(timerRef.current);
    setState("idle");
  }

  return (
    <>
      <form className="mb-6 w-full max-w-3xl" onSubmit={submit}>
        <label htmlFor="rag-query" className="sr-only">
          Ask SearchLens RAG
        </label>
        <div className="flex h-12 items-center rounded-full border border-[#dadad3] bg-[#f6f6f3] pr-1.5 pl-4 focus-within:border-[#91918c] focus-within:bg-white focus-within:shadow-[0_0_0_2px_#000,0_0_0_6px_#435ee5]">
          <svg
            aria-hidden="true"
            className="mr-2 size-5 shrink-0 text-[#91918c]"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            strokeWidth="2.2"
            strokeLinecap="round"
          >
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            ref={inputRef}
            id="rag-query"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Ask anything..."
            autoComplete="off"
            className="h-full min-w-0 flex-1 bg-transparent text-base text-black outline-none placeholder:text-[#91918c]"
          />
          <button
            type="submit"
            disabled={disabled}
            aria-label="Submit query"
            className="flex size-10 shrink-0 items-center justify-center rounded-full bg-[#e60023] text-white enabled:hover:bg-[#cc001f] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] disabled:cursor-not-allowed disabled:bg-[#f6f6f3] disabled:text-[#91918c]"
          >
            <svg
              aria-hidden="true"
              className="size-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 12h14m-7-7 7 7-7 7" />
            </svg>
          </button>
        </div>
        <div className="mt-2.5 flex flex-col gap-2 px-3 text-xs text-[#62625b] sm:flex-row sm:items-center sm:justify-between">
          <span>
            Press{" "}
            <kbd className="rounded border border-[#dadad3] bg-[#f6f6f3] px-1.5 py-0.5 font-mono text-[11px] text-black">
              Enter
            </kbd>{" "}
            to generate answers
          </span>
          <span className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-full bg-[#103c25]" />
            Verified Index Connected
          </span>
        </div>
      </form>

      {state === "idle" && (
        <div className="flex w-full max-w-3xl flex-wrap items-center justify-center gap-2 pt-2">
          <span className="mr-1 text-xs font-semibold tracking-wider text-[#91918c] uppercase">
            Try asking:
          </span>
          {suggestions.map(([label, value]) => (
            <button
              key={label}
              type="button"
              onClick={() => chooseSuggestion(value)}
              className="min-h-11 rounded-full bg-[#f6f6f3] px-3.5 text-xs font-semibold text-black hover:bg-[#dadad3] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]"
            >
              {label}
            </button>
          ))}
        </div>
      )}

      {state === "results" && results}

      {state === "loading" && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/50 p-4 backdrop-blur-[2px]">
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="loading-heading"
            className="relative flex w-full max-w-[440px] flex-col items-center rounded-[32px] bg-white p-8 text-center shadow-[0_16px_32px_-4px_rgba(0,0,0,0.12),0_8px_16px_-4px_rgba(0,0,0,0.08)]"
          >
            <button
              type="button"
              aria-label="Cancel search"
              onClick={cancel}
              className="absolute top-5 right-5 flex size-10 items-center justify-center rounded-full bg-[#f6f6f3] text-black hover:bg-[#e5e5e0] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]"
            >
              <svg aria-hidden="true" className="size-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2.5" strokeLinecap="round">
                <path d="m6 6 12 12M18 6 6 18" />
              </svg>
            </button>
            <Image
              src="/stitch/vector-book-logo.svg"
              alt=""
              width={112}
              height={112}
              className="mb-6 size-28"
            />
            <h2 id="loading-heading" className="mb-2 text-[22px] leading-[1.25] font-semibold text-black">
              Searching academic vectors...
            </h2>
            <p className="mb-6 text-base leading-[1.4] text-[#62625b]">
              This might take a few seconds.
            </p>
            <div className="mb-5 h-2 w-full overflow-hidden rounded-full bg-[#f6f6f3]" role="progressbar" aria-label="Searching">
              <div className="loading-progress h-full rounded-full bg-[#e60023]" />
            </div>
            <div className="inline-flex items-center gap-2 rounded-full bg-[#f6f6f3] px-3 py-1.5 text-xs text-[#62625b]" aria-live="polite">
              <span className="size-1.5 animate-ping rounded-full bg-[#e60023]" />
              Retrieving semantic chunks (embedding rank 0.94+)
            </div>
          </section>
        </div>
      )}
    </>
  );
}
