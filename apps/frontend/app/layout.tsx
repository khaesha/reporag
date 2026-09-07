import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "SearchLens RAG",
  description:
    "Search and synthesize knowledge with retrieval-augmented discovery.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className="antialiased">
      <body>{children}</body>
    </html>
  );
}
