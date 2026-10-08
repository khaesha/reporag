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

export function titleGroups(entries) {
  const grouped = new Map();
  for (const entry of entries) {
    const title = entry.title.trim();
    const normalized = title
      .normalize("NFKC")
      .toLocaleLowerCase("id-ID")
      .replace(/[^\p{L}\p{N}]+/gu, " ")
      .trim();
    const group = grouped.get(normalized) ?? [];
    group.push({ title, uri: entry.uri });
    grouped.set(normalized, group);
  }
  return [...grouped.values()]
    .map((group) => ({
      title: group.map((entry) => entry.title).toSorted((left, right) => left.localeCompare(right))[0],
      entries: group.toSorted((left, right) => left.uri.localeCompare(right.uri)),
    }))
    .toSorted((left, right) => left.title.localeCompare(right.title));
}

export async function verify(baseURL, entries) {
  const verified = [];
  const failures = [];
  for (const group of titleGroups(entries)) {
    const found = new Map();
    let failure = "";
    for (let page = 1; ; page++) {
    const url = new URL("/api/v1/search", baseURL);
      url.searchParams.set("q", group.title);
      url.searchParams.set("mode", "lexical");
      url.searchParams.set("limit", "10");
      url.searchParams.set("page", String(page));
      let response;
      let body;
      try {
        response = await fetch(url, { signal: AbortSignal.timeout(15_000) });
        body = await response.json();
      } catch (error) {
        failure = error.message || "request failed";
        break;
      }
      if (!response.ok) {
        failure = `HTTP ${response.status}`;
        break;
      }
      if (!Array.isArray(body.results) || !Number.isInteger(body.total) || body.total < 0) {
        failure = "invalid search response";
        break;
      }
      for (const [index, result] of body.results.entries()) {
        if (group.entries.some((entry) => entry.uri === result.uri)) {
          found.set(result.uri, { page, rank: (page - 1) * 10 + index + 1 });
        }
      }
      if (found.size === group.entries.length || page * 10 >= body.total) break;
    }
    for (const entry of group.entries) {
      const result = found.get(entry.uri);
      if (result) {
        verified.push({ ...entry, title_group_size: group.entries.length, disposition: "pass", ...result });
        continue;
      }
      verified.push({ ...entry, title_group_size: group.entries.length, disposition: "failure", reason: failure || "URI not found in paginated results" });
      failures.push(entry.uri);
    }
  }
  return { checked: entries.length, title_groups: titleGroups(entries).length, failures, passed: entries.length - failures.length, records: verified };
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
