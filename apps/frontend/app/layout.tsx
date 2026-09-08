import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "SearchLens",
  description: "Search thesis metadata and available abstracts.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className="antialiased">
      <body>{children}</body>
    </html>
  );
}
