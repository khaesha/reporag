import Image from "next/image";
import SearchForm from "./search-form";
import SearchResults from "./search-results";

export default function Home() {
  return (
    <div className="flex min-h-dvh flex-col bg-[#fbfbf9] text-[#33332e]">
      <header className="flex h-16 shrink-0 items-center justify-between border-b border-[#dadad3] bg-white px-4 sm:px-6 md:px-12">
        <div className="flex items-center gap-3">
          <Image
            src="/stitch/searchlens-logo.png"
            alt="SearchLens RAG logo"
            width={36}
            height={36}
            priority
            className="size-9 rounded-lg border border-[#dadad3] bg-white object-contain p-0.5"
          />
          <span className="text-lg font-bold tracking-tight text-black">
            SearchLens RAG
          </span>
        </div>
        <div className="flex items-center gap-3">
          <span className="hidden rounded-full bg-[#f6f6f3] px-3 py-1 text-xs font-semibold text-[#62625b] md:inline-block">
            Knowledge Base Active
          </span>
          <button
            type="button"
            className="min-h-11 rounded-2xl bg-[#e60023] px-4 text-sm font-bold text-white hover:bg-[#cc001f] focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5]"
          >
            Sign up
          </button>
        </div>
      </header>

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col items-center justify-center px-4 py-12 sm:px-6">
        <section
          className="mb-6 max-w-2xl space-y-2 text-center"
          aria-labelledby="page-title"
        >
          <div className="inline-flex items-center gap-2 rounded-full border border-[#dadad3] bg-[#f6f6f3] px-3.5 py-1.5 text-xs font-semibold text-[#211922]">
            <span className="size-2 animate-pulse rounded-full bg-[#e60023]" />
            Retrieval-Augmented Discovery
          </div>
          <h1
            id="page-title"
            className="text-4xl font-bold tracking-tight text-black md:text-5xl"
          >
            Ask anything across knowledge.
          </h1>
          <p className="text-base leading-relaxed text-[#62625b]">
            Synthesize multimodal sources, documents, and visual ideas with
            enterprise-grade retrieval.
          </p>
        </section>

        <SearchForm results={<SearchResults />} />
      </main>

      <footer className="border-t border-[#dadad3] bg-white px-6 py-5 text-center text-xs text-[#62625b]">
        Crafted with Pinterest Design System specifications • Inter Display •
        {" {rounded.full} Pill Geometry"}
      </footer>
    </div>
  );
}
