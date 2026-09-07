const results = [
  ["Optimal Chunk Sizing & Sliding Window Overlap in Enterprise RAG Systems", "By Dr. Aris Thorne • Updated 2 days ago • AI Architecture", "98%", "Empirical evaluation comparing 256, 512, and 1024 token chunk sizes with recursive character boundary splitting. Demonstrates how a 15% sliding window overlap preserves multi-sentence context and resolves coreference resolution gaps in sparse document retrieval pipelines."],
  ["Hybrid Retrieval: Merging Dense Vector Embeddings with BM25 Sparse Indexing", "By Vector Systems Team • Updated yesterday • Search Infrastructure", "96%", "A production implementation of reciprocal rank fusion (RRF) marrying HNSW dense k-NN vector indexes with term-frequency BM25 scores. Fixes exact keyword domain misses while retaining conceptual semantic matching across multimodal documents."],
  ["Two-Stage RAG: Cross-Encoder Re-Ranking Architecture & Latency Benchmarks", "By Research Engineering • Updated 4 days ago • Model Optimization", "94%", "Assessing latency tradeoffs when filtering top-100 bi-encoder candidates through a quantized MiniLM cross-encoder. Results demonstrate a 24% boost in MRR@10 with only 18ms p95 latency overhead on clustered inference nodes."],
  ["Hierarchical Document Chunking with Parent-Child Vector Associations", "By Maya Lin, Staff ML Scientist • Updated 5 days ago • Data Ingestion", "91%", "Index micro-chunks (128 tokens) for hyper-precise vector discovery while passing larger parent segments (2048 tokens) to the generative context window. Eliminates truncation artifacts while ensuring fine-grained vector similarity matches."],
  ["Hallucination Guardrails: Grounding Evaluation Metrics via RAGAS & TruLens", "By Quality Assurance Guild • Updated 1 week ago • Evaluation Frameworks", "89%", "Automated evaluation workflows for measuring context relevance, answer groundedness, and hallucination frequency. Integrates synthetic test set generation across multi-tenant knowledge bases with automated citation attribution scores."],
  ["Token Budget Management & Context Compression Strategies for LLM Prompts", "By Platform Scalability Group • Updated 1 week ago • Cost & Performance", "87%", "Techniques for dynamic prompt context pruning using semantic sentence encoders and extractive summarization. Achieves 40% reduction in generation token costs without compromising factuality or answer completeness in domain QA benchmarks."],
  ["Multimodal RAG: Image-Text Interleaved Ingestion & CLIP Vector Alignment", "By Visual Intelligence Team • Updated 2 weeks ago • Vision & Multimodal", "85%", "Strategies for parsing complex PDF diagrams, flowcharts, and technical tables into joint vector spaces. Combines OCR extraction, layout detection transformers, and dual-projection embedding spaces for cohesive cross-modal question answering."],
  ["Adaptive Query Rewriting & HyDE (Hypothetical Document Embeddings) Analysis", "By Cognitive Retrieval Lab • Updated 2 weeks ago • Query Engineering", "83%", "A deep dive into zero-shot query expansion where an LLM generates speculative answers prior to vector search. Explores trade-offs in hallucination propagation versus recall improvement for ambiguous and shorthand developer search queries."],
  ["GraphRAG: Augmenting Vector Retrieval with Knowledge Graph Traversal", "By Knowledge Systems Group • Updated 3 weeks ago • Graph Architectures", "81%", "Overcoming k-NN retrieval blindness in complex multi-hop reasoning by combining entity relationship graph communities with dense embeddings. Enables deep inductive summarization across disconnected institutional datasets."],
  ["Vector Database Sharding & Distributed HNSW Index Maintenance at Scale", "By Cloud Infrastructure Core • Updated 1 month ago • Database Engineering", "79%", "Operational principles for updating live HNSW vector indexes without query degradation. Covers partition clustering, incremental centroid recalculation, and memory-mapped file techniques for hundred-million vector collections."],
] as const;

export default function SearchResults() {
  return (
    <section className="mt-8 w-full max-w-3xl text-left" aria-label="Search results">
      <div className="mb-4 flex flex-col gap-3 border-b border-[#dadad3] py-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <span className="text-sm font-bold text-[#62625b]">Sort by:</span>
          <span className="text-xs font-medium text-[#91918c]">(10 results)</span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {["Similarity Score", "Alphabetical", "Date"].map((label, index) => (
            <button
              key={label}
              type="button"
              className={`min-h-11 rounded-full px-4 text-sm font-bold focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] ${
                index === 0
                  ? "bg-black text-white"
                  : "bg-[#f6f6f3] text-black hover:bg-[#dadad3]"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {results.map(([title, meta, match, description]) => (
          <article key={title} className="rounded-2xl border border-[#dadad3] bg-white p-5 hover:border-[#91918c]">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
              <div>
                <h2 className="text-lg leading-snug font-semibold text-black">{title}</h2>
                <p className="mt-1 text-sm text-[#62625b]">{meta}</p>
              </div>
              <span className="w-fit shrink-0 rounded-full border border-[#dadad3] bg-[#f6f6f3] px-3 py-1 text-xs font-semibold text-[#e60023]">
                {match} Match
              </span>
            </div>
            <p className="mt-2.5 text-base leading-[1.4] text-[#33332e]">{description}</p>
          </article>
        ))}
      </div>

      <nav aria-label="Search result pages" className="mt-8 mb-12 flex flex-wrap items-center justify-center gap-2">
        {["Previous", "1", "2", "3", "4", "Next"].map((label) => (
          <button
            key={label}
            type="button"
            aria-current={label === "1" ? "page" : undefined}
            className={`min-h-11 rounded-full text-sm font-bold focus-visible:outline-4 focus-visible:outline-offset-2 focus-visible:outline-[#435ee5] ${
              label === "1"
                ? "min-w-11 bg-black text-white"
                : "bg-[#e5e5e0] px-4 text-black hover:bg-[#c8c8c1]"
            }`}
          >
            {label}
          </button>
        ))}
      </nav>
    </section>
  );
}
