# HNSW Lab — frontend

React 19 + TypeScript (Vite 8), Tailwind CSS 4, TanStack Query, React Router, zod, MSW, Vitest.

## Commands

```bash
npm install
npm run dev          # dev server on :5173, proxies /api to :8080
npm run dev:mocks    # dev server with MSW mocks, no backend needed
npm run lint         # ESLint
npm run test         # Vitest
npm run build        # tsc -b + vite build
npm run format       # Prettier
```

## Layout

```
src/
  app/         router, shell, theme, page header
  api/         fetch client, zod schemas, query hooks
  components/  hand-written UI primitives
  features/    datasets, index, search, benchmark pages
  mocks/       MSW handlers and in-memory state
  styles/      Tailwind entry and design tokens
  test/        Vitest setup and render helper
```

The API contract lives in `../docs/API.md`, the plan in `../docs/PLAN.md`.
