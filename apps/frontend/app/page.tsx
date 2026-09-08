import Image from "next/image";
import SearchForm from "./search-form";

export default function Home() {
  return (
    <div className="flex min-h-dvh flex-col bg-[#fbfbf9] text-[#33332e]">
      <header className="flex h-16 shrink-0 items-center justify-between border-b border-[#dadad3] bg-white px-4 sm:px-6 md:px-12">
        <div className="flex items-center gap-3">
          <Image
            src="/stitch/searchlens-logo.png"
            alt="SearchLens logo"
            width={36}
            height={36}
            priority
            className="size-9 rounded-lg border border-[#dadad3] bg-white object-contain p-0.5"
          />
          <span className="text-lg font-bold tracking-tight text-black">
            SearchLens
          </span>
        </div>
        <span className="rounded-full bg-[#f6f6f3] px-3 py-1.5 text-xs font-semibold text-[#62625b]">
          Chapter 1
        </span>
      </header>

      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col items-center px-4 py-12 sm:px-6">
        <section
          className="mb-6 max-w-2xl space-y-2 text-center"
          aria-labelledby="page-title"
        >
          <div className="inline-flex items-center gap-2 rounded-full border border-[#dadad3] bg-[#f6f6f3] px-3.5 py-1.5 text-xs font-semibold text-[#211922]">
            UPI Repository Discovery
          </div>
          <h1
            id="page-title"
            className="text-4xl font-bold tracking-tight text-black md:text-5xl"
          >
            Search thesis metadata and available abstracts
          </h1>
          <p className="text-base leading-relaxed text-[#62625b]">
            Find theses by title, author, topic, year, degree program, or item
            type, then open the authoritative repository record.
          </p>
        </section>

        <SearchForm />
      </main>

      <footer className="border-t border-[#dadad3] bg-white px-6 py-5 text-center text-xs text-[#62625b]">
        SearchLens • Thesis metadata discovery • UPI Repository sources
      </footer>
    </div>
  );
}
