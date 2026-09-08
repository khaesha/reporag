# SearchLens frontend

## Local development

Start PostgreSQL, import the corpus, and run the backend from the repository root:

```sh
make db-up
make import
make api
```

Create the public frontend environment once, then start Next.js:

```sh
cd apps/frontend
cp .env.local.example .env.local
npm run dev
```

`NEXT_PUBLIC_API_URL` is the browser-visible backend origin and is fixed when Next.js builds the app. Set it before `npm run build`. Never put database credentials in this file.

Open [http://localhost:3000](http://localhost:3000). Search requests go directly to the backend, whose `FRONTEND_ORIGIN` must allow this frontend origin.

## Checks

```sh
npm test
npm run lint
npx tsc --noEmit
npm exec -- next build --webpack
```
