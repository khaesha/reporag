import { readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

async function files(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(entries.map(async entry => entry.isDirectory() ? files(path.join(directory, entry.name)) : [/^repository_.*_data\.json$/.test(entry.name) ? path.join(directory, entry.name) : null]));
  return nested.flat().filter(Boolean).sort();
}

export async function records(directory, quarantinePath) {
  const quarantined = new Set((JSON.parse(await readFile(quarantinePath))).map(item => item.uri));
  const data = await Promise.all((await files(directory)).map(file => readFile(file, "utf8").then(JSON.parse)));
  const seen = new Set();
  return data.flat().filter(item => {
    const uri = item?.uri?.trim();
    if (!item?.title?.trim() || !uri || quarantined.has(uri) || seen.has(uri)) return false;
    seen.add(uri);
    return true;
  });
}

export async function verify(baseURL, entries) {
  const failures = [];
  for (let offset = 0; offset < entries.length; offset += 20) {
    const batch = await Promise.all(entries.slice(offset, offset + 20).map(async entry => {
    const url = new URL("/api/v1/search", baseURL);
    url.searchParams.set("q", entry.title);
    url.searchParams.set("mode", "lexical");
    url.searchParams.set("limit", "10");
    const response = await fetch(url, { signal: AbortSignal.timeout(15_000) });
    const body = await response.json();
    return !response.ok || !body.results?.some(result => result.uri === entry.uri) ? entry.uri : null;
    }));
    failures.push(...batch.filter(Boolean));
  }
  return { checked: entries.length, failures, passed: entries.length - failures.length };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  const root = path.resolve(path.dirname(script), "..");
  const [baseURL = "http://localhost:8080", output] = process.argv.slice(2);
  verify(baseURL, await records(path.join(root, "docs/repository-data"), path.join(root, "docs/chapter-2/quarantine.json"))).then(async result => {
    const text = `${JSON.stringify(result, null, 2)}\n`;
    if (output) await writeFile(path.resolve(output), text);
    console.log(text.trim());
    if (result.failures.length) process.exitCode = 1;
  }).catch(error => { console.error(error.message); process.exitCode = 1; });
}
