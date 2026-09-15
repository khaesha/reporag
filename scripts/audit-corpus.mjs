import { createHash } from "node:crypto";
import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const fields = [
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

const optionalStrings = new Set([
  "abstract",
  "item_type",
  "subjects",
  "divisions",
  "depositing_user",
  "date_deposited",
]);
const months = new Map(
  ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"].map(
    (month, index) => [month, index],
  ),
);

const missing = (value) =>
  value == null ||
  (typeof value === "string" && value.trim() === "") ||
  (Array.isArray(value) && value.length === 0);
const increment = (map, key) => map.set(key, (map.get(key) ?? 0) + 1);
const counts = (map) =>
  [...map]
    .map(([value, count]) => ({ value, count }))
    .sort((a, b) => a.value.localeCompare(b.value));
const normalizedTitle = (title) =>
  title
    .normalize("NFKC")
    .toLocaleLowerCase("id-ID")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();

const validDate = (value) => {
  const match = /^(\d{2}) ([A-Z][a-z]{2}) (\d{4}) (\d{2}):(\d{2})$/.exec(value);
  if (!match || !months.has(match[2])) return false;
  const [, day, month, year, hour, minute] = match;
  const date = new Date(Date.UTC(+year, months.get(month), +day, +hour, +minute));
  return (
    date.getUTCFullYear() === +year &&
    date.getUTCMonth() === months.get(month) &&
    date.getUTCDate() === +day &&
    date.getUTCHours() === +hour &&
    date.getUTCMinutes() === +minute
  );
};

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

async function loadRules() {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const [normalization, quarantine] = await Promise.all([
    readFile(path.join(root, "docs/chapter-2/normalization.json"), "utf8").then(JSON.parse),
    readFile(path.join(root, "docs/chapter-2/quarantine.json"), "utf8").then(JSON.parse),
  ]);
  return { normalization, quarantine };
}

function invalidValues(record) {
  const invalid = [];
  for (const field of ["title", "uri"]) {
    if (typeof record[field] !== "string" || record[field].trim() === "") {
      invalid.push({ field, message: "must be a non-empty string" });
    }
  }
  if (!Array.isArray(record.authors) || record.authors.some((author) => typeof author !== "string")) {
    invalid.push({ field: "authors", message: "must be an array of strings" });
  }
  for (const field of optionalStrings) {
    if (record[field] != null && typeof record[field] !== "string") {
      invalid.push({ field, message: "must be a string or null" });
    }
  }
  if (typeof record.date_deposited === "string" && record.date_deposited.trim() && !validDate(record.date_deposited.trim())) {
    invalid.push({ field: "date_deposited", message: "must match DD Mon YYYY HH:mm" });
  }
  return invalid;
}

export async function auditCorpus(directory, suppliedRules) {
  const { normalization, quarantine } = suppliedRules ?? (await loadRules());
  const quarantineByURI = new Map(quarantine.map((item) => [item.uri, item.reason]));
  const files = await findCorpusFiles(directory);
  const digest = createHash("sha256");
  const missingFields = Object.fromEntries(fields.map((field) => [field, 0]));
  const divisionCounts = new Map();
  const normalizedDivisionCounts = new Map();
  const itemTypeCounts = new Map();
  const yearCounts = new Map();
  const uriCounts = new Map();
  const titleGroups = new Map();
  const unmappedSubjects = new Map();
  const invalidFields = [];
  const quarantinedURIs = new Map();
  let records = 0;

  for (const file of files) {
    const contents = await readFile(file);
    const relative = path.relative(directory, file).split(path.sep).join("/");
    digest.update(relative).update("\0").update(contents).update("\0");
    let data;
    try {
      data = JSON.parse(contents);
    } catch (error) {
      throw new Error(`${file}: ${error.message}`);
    }
    if (!Array.isArray(data)) throw new Error(`${file}: expected a JSON array`);

    const yearMatch = /(?:^|_)(\d{4})_data\.json$/.exec(path.basename(file));
    const sourceYear = yearMatch ? Number(yearMatch[1]) : null;
    if (sourceYear == null || sourceYear < 1900 || sourceYear > 2100) {
      invalidFields.push({ file: relative, index: null, field: "source_year", message: "filename must contain a year from 1900 to 2100" });
    }

    for (const [index, record] of data.entries()) {
      records++;
      if (!record || typeof record !== "object" || Array.isArray(record)) {
        invalidFields.push({ file: relative, index, field: "record", message: "must be an object" });
        continue;
      }
      for (const field of fields) if (missing(record[field])) missingFields[field]++;
      for (const invalid of invalidValues(record)) invalidFields.push({ file: relative, index, ...invalid });
      if (sourceYear != null) increment(yearCounts, String(sourceYear));
      if (typeof record.divisions === "string" && record.divisions.trim()) increment(divisionCounts, record.divisions.trim());
      if (typeof record.item_type === "string" && record.item_type.trim()) increment(itemTypeCounts, record.item_type.trim());
      if (typeof record.uri !== "string" || !record.uri.trim()) continue;

      const uri = record.uri.trim();
      increment(uriCounts, uri);
      if (quarantineByURI.has(uri)) {
        quarantinedURIs.set(uri, quarantineByURI.get(uri));
        continue;
      }
      const division = normalization.divisions[record.divisions?.trim()];
      if (division) increment(normalizedDivisionCounts, division);
      if (typeof record.title === "string" && record.title.trim()) {
        const key = normalizedTitle(record.title);
        const group = titleGroups.get(key) ?? { title: record.title.trim(), uris: new Set() };
        group.uris.add(uri);
        titleGroups.set(key, group);
      }
      if (typeof record.subjects === "string" && record.subjects.trim()) {
        const subject = record.subjects.toLocaleLowerCase("en");
        const recognized = Object.keys(normalization.subject_classes).some((prefix) => subject.includes(prefix.toLocaleLowerCase("en")));
        if (!recognized || record.subjects.trim().startsWith(">")) increment(unmappedSubjects, record.subjects.trim());
      }
    }
  }

  const duplicateUris = [...uriCounts]
    .filter(([, count]) => count > 1)
    .map(([uri, count]) => ({ uri, count }))
    .sort((a, b) => a.uri.localeCompare(b.uri));
  const probableDuplicateTitles = [...titleGroups.values()]
    .filter(({ uris }) => uris.size > 1)
    .map(({ title, uris }) => ({ title, uris: [...uris].sort() }))
    .sort((a, b) => a.title.localeCompare(b.title));
  const quarantined = [...quarantinedURIs]
    .map(([uri, reason]) => ({ uri, reason }))
    .sort((a, b) => a.uri.localeCompare(b.uri));

  return {
    corpus_sha256: digest.digest("hex"),
    files: files.length,
    records,
    unique_uris: uriCounts.size,
    valid_unique_uris: uriCounts.size - quarantined.length,
    duplicate_uri_count: duplicateUris.reduce((total, item) => total + item.count - 1, 0),
    quarantined_uri_count: quarantined.length,
    yearly_coverage: counts(yearCounts),
    degree_programs: counts(divisionCounts),
    normalized_degree_programs: counts(normalizedDivisionCounts),
    item_types: counts(itemTypeCounts),
    missing_fields: missingFields,
    invalid_field_count: invalidFields.length,
    invalid_fields: invalidFields,
    duplicate_uris: duplicateUris,
    probable_duplicate_title_count: probableDuplicateTitles.length,
    probable_duplicate_titles: probableDuplicateTitles,
    unmapped_subject_value_count: unmappedSubjects.size,
    unmapped_subjects: counts(unmappedSubjects),
    quarantined_uris: quarantined,
  };
}

const script = fileURLToPath(import.meta.url);
if (process.argv[1] && path.resolve(process.argv[1]) === script) {
  if (process.argv.length > 4) {
    console.error("usage: node scripts/audit-corpus.mjs [corpus-directory] [output-file]");
    process.exitCode = 2;
  } else {
    const directory = process.argv[2]
      ? path.resolve(process.argv[2])
      : path.resolve(path.dirname(script), "../docs/repository-data");
    try {
      const output = `${JSON.stringify(await auditCorpus(directory), null, 2)}\n`;
      if (process.argv[3]) await writeFile(path.resolve(process.argv[3]), output);
      console.log(output.trimEnd());
    } catch (error) {
      console.error(error.message);
      process.exitCode = 1;
    }
  }
}
