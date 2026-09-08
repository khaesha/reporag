/**
 * @param {string} apiURL
 * @param {{
 *   query: string,
 *   page: number,
 *   limit: number,
 *   sort: string,
 *   year?: string,
 *   division?: string,
 *   itemType?: string,
 *   hasAbstract?: string,
 * }} params
 */
export function buildSearchURL(apiURL, params) {
  const base = apiURL.trim().replace(/\/+$/, "");
  if (!base) throw new Error("NEXT_PUBLIC_API_URL is not configured");

  const url = new URL(`${base}/api/v1/search`);
  url.searchParams.set("q", params.query);
  url.searchParams.set("page", String(params.page));
  url.searchParams.set("limit", String(params.limit));
  url.searchParams.set("sort", params.sort);
  if (params.year) url.searchParams.set("year", params.year);
  if (params.division) url.searchParams.set("division", params.division);
  if (params.itemType) url.searchParams.set("item_type", params.itemType);
  if (params.hasAbstract)
    url.searchParams.set("has_abstract", params.hasAbstract);
  return url.toString();
}
