You are operating as an autonomous scraping assistant using `playwright-mcp` tools and local file operations.

**Task:** Extract thesis/item metadata from the target EPrints repository for a given year and append the structured data into a JSON file.

**Input Configuration:**
- START_URL:"[https://repository.upi.edu/view/divisions/ILKOM/2020.html](https://repository.upi.edu/view/divisions/ILKOM/2020.html)"
- OUTPUT_FILE: "repository_2020_data.json"
- QUEUE_FILE: "url_queue.json"

---

### Step 1: Initial Discovery & Queueing
1. Use `playwright-mcp` to open `START_URL`.
2. Extract all anchor (`<a>`) tags pointing to individual item detail pages (e.g., links inside the list items containing thesis titles).
3. Extract the absolute URLs for all items and save them as a JSON array inside `QUEUE_FILE`. 
4. Log the total number of extracted links to stdout.

---

### Step 2: Batch Processing Loop
Process the URLs from `QUEUE_FILE` in **batches of 5 items** at a time:

1. Read `QUEUE_FILE` and pick the next 5 unprocessed URLs.
2. For each URL in the current batch:
   a. Navigate to the page using `playwright-mcp`.
   b. Extract the following structured fields:
      - `title` (text from main header h1)
      - `abstract` (text under Abstract section)
      - `authors` (author names under header)
      - `item_type` (from metadata table, e.g., "Thesis (S1)")
      - `subjects` (from metadata table)
      - `divisions` (from metadata table)
      - `depositing_user` (from metadata table)
      - `date_deposited` (from metadata table)
      - `uri` (from metadata table)
   c. If a field is missing, set its value to `null`.
3. Load `OUTPUT_FILE` (if it exists, read array; if not, initialize empty `[]`), append the newly scraped item objects, and write the updated array back to `OUTPUT_FILE`.
4. Remove the 5 processed URLs from `QUEUE_FILE`.

---

### Rules & Safety Guardrails:
- **No Browser History Back Navigation:** Always navigate directly to the target item URL using `playwright-mcp`.
- **Atomic File Updates:** Save the updated JSON array to disk after every batch of 5 items to prevent data loss.
- **Completion Check:** Repeat Step 2 until `QUEUE_FILE` is empty, then delete `QUEUE_FILE` and inform me that extraction is complete.
