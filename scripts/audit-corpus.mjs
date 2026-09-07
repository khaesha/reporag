import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const fields = [
  "title",
  "abstract",
  "authors",
  "item_type",
  "subjects",
  "divisions",
  "depositing_user",
  "date_deposited",
  "uri",
];

const missing = (value) =>
  value == null ||
  (typeof value === "string" && value.trim() === "") ||
  (Array.isArray(value) && value.length === 0);

export async function findCorpusFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries.sort((a, b) => a.name.localeCompare(b.name))) {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...(await findCorpusFiles(target)));
    else if (/^repository_.*_data\.json$/.test(entry.name)) files.push(target);
  }

  return files;
}

export async function auditCorpus(directory) {
  const files = await findCorpusFiles(directory);
  const missingFields = Object.fromEntries(fields.map((field) => [field, 0]));
  const divisionCounts = new Map();
  const uriCounts = new Map();
  let records = 0;

  for (const file of files) {
    let data;
    try {
      data = JSON.parse(await readFile(file, "utf8"));
    } catch (error) {
      throw new Error(`${file}: ${error.message}`);
    }
    if (!Array.isArray(data)) throw new Error(`${file}: expected a JSON array`);

    for (const [index, record] of data.entries()) {
      if (!record || typeof record !== "object" || Array.isArray(record)) {
        throw new Error(`${file}[${index}]: expected an object`);
      }
      records++;
      for (const field of fields) if (missing(record[field])) missingFields[field]++;
      if (!missing(record.divisions)) {
        divisionCounts.set(record.divisions, (divisionCounts.get(record.divisions) ?? 0) + 1);
      }
      if (!missing(record.uri)) uriCounts.set(record.uri, (uriCounts.get(record.uri) ?? 0) + 1);
    }
  }

  const duplicateUris = [...uriCounts]
    .filter(([, count]) => count > 1)
    .map(([uri, count]) => ({ uri, count }))
    .sort((a, b) => a.uri.localeCompare(b.uri));

  return {
    files: files.length,
    records,
    unique_uris: uriCounts.size,
    duplicate_uri_count: duplicateUris.reduce((total, item) => total + item.count - 1, 0),
    degree_programs: [...divisionCounts]
      .map(([value, count]) => ({ value, count }))
      .sort((a, b) => a.value.localeCompare(b.value)),
    missing_fields: missingFields,
    duplicate_uris: duplicateUris,
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 3) {
    console.error("usage: node scripts/audit-corpus.mjs [corpus-directory]");
    process.exitCode = 2;
  } else {
    const directory = process.argv[2]
      ? path.resolve(process.argv[2])
      : path.resolve(path.dirname(script), "../docs/repository-data");
    try {
      console.log(JSON.stringify(await auditCorpus(directory), null, 2));
    } catch (error) {
      console.error(error.message);
      process.exitCode = 1;
    }
  }
}
