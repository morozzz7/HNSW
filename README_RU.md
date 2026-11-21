# HNSW Coursework

![Go 1.22+](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![TypeScript 6](https://img.shields.io/badge/TypeScript-6-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![React 19](https://img.shields.io/badge/React-19-61DAFB?style=for-the-badge&logo=react&logoColor=white)
![Vite 8](https://img.shields.io/badge/Vite-8-646CFF?style=for-the-badge&logo=vite&logoColor=white)
![Tailwind CSS 4](https://img.shields.io/badge/Tailwind-4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)

![TanStack Query 5](https://img.shields.io/badge/TanStack%20Query-5-FF4154?style=for-the-badge&logo=tanstack&logoColor=white)
![Sigma.js 3](https://img.shields.io/badge/Sigma.js-3-5B8FF9?style=for-the-badge)
![HNSW с нуля](https://img.shields.io/badge/HNSW-from%20scratch-5BE0B5?style=for-the-badge)
![CI проходит](https://img.shields.io/badge/CI-passing-2EA043?style=for-the-badge&logo=githubactions&logoColor=white)
![Docker Compose](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)

[English](README.md) · **Русский**

Проектирование структуры данных HNSW и её применение для приближённого поиска ближайших соседей.

## Статус

Все запланированные этапы завершены: ядро HNSW, загрузчики и генераторы датасетов, точный поиск
и метрики качества, модуль бенчмарка, HTTP API, укрепление бэкенда и React-интерфейс (датасеты,
построение индекса, поиск, визуализация графа, бенчмарк). Этап 9 охватывает сквозной сценарий,
граничные случаи, производительность графа, Docker Compose и документацию.

## Архитектура

### Сервис

```mermaid
flowchart LR
  subgraph Client["Браузер · :5173"]
    Pages["Страницы React<br/>Датасеты · Построение · Поиск · Бенчмарк"]
    Hooks["TanStack Query<br/>валидация zod"]
  end

  subgraph Edge["Vite dev-сервер · :5173"]
    Proxy["/api → :8080<br/>VITE_API_PROXY_TARGET"]
    Mock["MSW-моки<br/>VITE_USE_MOCKS"]
  end

  subgraph HTTP["Go HTTP API · :8080"]
    Routes["net/http<br/>CORS · ошибки · SSE"]
    Manager["indexmanager<br/>состояние сборки · внешние ID · RWMutex"]
  end

  subgraph Core["Ядро"]
    HNSW["hnsw<br/>граф и поиск"]
    Brute["bruteforce<br/>точный k-NN"]
    Data["dataset + projection<br/>загрузчики · PCA"]
    Bench["benchmark<br/>сетка · метрики · экспорт"]
  end

  subgraph Storage["Данные"]
    Files["CSV / JSON / .fvecs"]
    Memory["Хранилище в памяти · 1 ГиБ"]
    Snapshot["Снимок индекса · *.idx"]
  end

  Pages --> Hooks --> Proxy
  Mock -.-> Hooks
  Proxy --> Routes --> Manager
  Manager --> HNSW
  Manager --> Brute
  Manager --> Data
  Manager --> Bench
  Data --> Files
  Data --> Memory
  Manager --> Snapshot
  Bench --> HNSW
  Bench --> Brute
  Routes -- "SSE: прогресс / готово / ошибка" --> Pages

  click Pages "docs/screenshots/03-search-graph.png" "Скриншот интерфейса"
  click Proxy "frontend/vite.config.ts" "Настройка прокси Vite"
  click Routes "docs/API.md" "Контракт HTTP API"
  click Manager "backend/internal/indexmanager" "Исходный код менеджера индекса"
  click HNSW "backend/internal/hnsw" "Исходный код ядра HNSW"
  click Brute "backend/internal/bruteforce" "Исходный код полного перебора"
  click Data "backend/internal/dataset" "Исходный код данных"
  click Bench "backend/internal/benchmark" "Исходный код бенчмарка"
```

### Алгоритм HNSW

```mermaid
flowchart TB
  subgraph Graph["Многослойная структура"]
    Entry["L2 · входная точка"]
    Mid["L1 · разреженные длинные связи"]
    Base["L0 · все векторы"]
    Entry --> Mid --> Base
  end

  subgraph InsertFlow["ВСТАВКА — алгоритм 1"]
    I1["Уровень: floor(-ln(U) · mL)"]
    I2["SEARCH-LAYER от входной точки"]
    I3["SELECT-NEIGHBORS-HEURISTIC"]
    I4["Двунаправленные связи + обрезка при переполнении"]
    I1 --> I2 --> I3 --> I4
  end

  subgraph SearchFlow["K-NN ПОИСК — алгоритм 5"]
    S1["Старт от входной точки на maxLevel"]
    S2["Жадный спуск: по одной ближайшей вершине на слой"]
    S3["SEARCH-LAYER на L0 с ef = max(efSearch, k)"]
    S4["Возврат k ближайших по расстоянию"]
    S5["Recall@k относительно полного перебора"]
    S1 --> S2 --> S3 --> S4 --> S5
  end

  subgraph Primitives["Общие примитивы"]
    Dist["квадрат L2 / косинус / скалярное произведение"]
    Visited["посещённые: generation + sync.Pool"]
    Heap["типизированные кучи min/max"]
    Select["простой / эвристический отбор соседей"]
  end

  Entry --> I2
  Base --> I4
  Entry --> S1
  Base --> S3
  Dist --> S3
  Visited --> S2
  Heap --> S3
  Select --> I3

  click I1 "backend/internal/hnsw/index.go" "Генерация уровня"
  click I2 "backend/internal/hnsw/insert.go" "Вставка и вызовы SEARCH-LAYER"
  click I3 "backend/internal/hnsw/neighbors.go" "Алгоритмы отбора соседей"
  click S2 "backend/internal/hnsw/search.go" "SEARCH-LAYER и K-NN поиск"
  click Dist "backend/internal/hnsw/distance.go" "Метрики расстояний"
  click Visited "backend/internal/hnsw/visited.go" "Множество посещённых"
  click Heap "backend/internal/hnsw/heap.go" "Типизированные кучи"
  click S5 "backend/internal/bruteforce" "Точный поиск для сравнения"
```

## Стек

- **Бэкенд:** Go 1.22+, модуль `hnsw-coursework`, HNSW с нуля (без сторонних ANN-библиотек), стандартный `net/http`.
- **Фронтенд:** React 19 + TypeScript (Vite 8), Tailwind CSS 4, TanStack Query 5, React Router 7, zod, MSW, Vitest, Sigma.js v3 + Graphology.
- **Векторы:** `[]float32`; внутренние ID: `uint32`; внутри используется квадрат L2 (`sqrt` только на выходе API).

## Структура репозитория

```
README_RU.md       Русская версия этого документа
backend/            Go-бэкенд: cmd/server, internal/{hnsw,bruteforce,dataset,projection,benchmark,api,indexmanager}
frontend/           React + TypeScript (Vite) интерфейс
docs/               PLAN.md, API.md, screenshots/
scripts/            скрипты загрузки датасетов (сами датасеты не коммитятся)
.github/workflows/  GitHub Actions CI
docker-compose.yml  локальный запуск бэкенда и фронтенда в контейнерах
```

## Запуск

Бэкенд (слушает `:8080`, порт можно изменить через `PORT`):

```bash
cd backend
make run        # go run ./cmd/server
```

Фронтенд (слушает `:5173`, проксирует `/api` на `http://localhost:8080`):

```bash
cd frontend
npm install
npm run dev          # с реальным бэкендом
npm run dev:mocks    # с MSW-моками, без бэкенда
npm run build        # проверка типов + production-сборка
```

Из корня репозитория цели бэкенда доступны как `make -C backend <цель>`.

Docker Compose (оба сервиса, фронтенд на :5173, бэкенд на :8080):

```bash
docker-compose up --build
docker-compose down
```

В контейнерах адрес прокси Vite берётся из `VITE_API_PROXY_TARGET` (по умолчанию
`http://localhost:8080`); Compose задаёт `http://backend:8080`.

## Сценарий работы

1. **Датасеты** (`/datasets`) — загрузка `.csv`/`.json` через drag-and-drop или генерация
   синтетического набора (uniform, gaussian, clusters) с фиксированным seed; превью и статистика
   координат.

   ![Список датасетов и превью](docs/screenshots/01-datasets.png)

2. **Построение** (`/index`) — выбор датасета, `M`, `efConstruction`, метрики, эвристики отбора
   соседей и seed. Прогресс приходит по SSE; после готовности показываются число вершин по уровням
   и гистограмма степеней.

   ![Сборка индекса и статистика графа](docs/screenshots/02-index.png)

3. **Поиск** (`/search`) — запрос по ID вектора или по координатам, настройка `k`, `efSearch` и режим
   HNSW / полный перебор / оба. Таблица результатов показывает Recall@k, пересечение и ускорение;
   WebGL-граф поддерживает переключение слоёв, укладки PCA и ForceAtlas2, просмотр вершины, режим
   окрестности и пошаговую анимацию спуска.

   ![Результаты поиска и визуализация графа](docs/screenshots/03-search-graph.png)

4. **Бенчмарк** (`/benchmark`) — прогон сетки `M × efConstruction × efSearch` и графики recall,
   Парето по QPS, перцентилей задержки, времени построения и таблица результатов; графики
   экспортируются в PNG, таблица — в CSV.

   ![Графики бенчмарка и результаты](docs/screenshots/04-benchmark.png)

## Проверка

```bash
cd backend && gofmt -l . && go vet ./... && go test ./... && go test -race ./...
cd frontend && npm run format:check && npm run lint && npm run test && npm run build
```

Полный сценарий также прогонялся в Chromium с реальным бэкендом: загрузка CSV и генерация →
сборка индекса → поиск (включая `k > N` и rebuild) → граф → бенчмарк. Граф проверен на 2000 вершин
и ~17 тыс. рёбер без ошибок в консоли.

Контракт API: `docs/API.md`. План и прогресс: `docs/PLAN.md`.

## Примечания

- Большие датасеты (SIFT и др.) никогда не коммитятся; скачивайте их через `scripts/download_sift.sh`.
  По HTTP принимаются только `.csv` и `.json`; `.fvecs`/`.ivecs` читаются загрузчиками Go и бенчмарками.
- CORS включён для `http://localhost:5173`; Vite-сервер также проксирует `/api`, поэтому интерфейс
  не зависит от CORS.
- Хранилище датасетов в памяти ограничено 1 ГиБ; при превышении возвращается `507 STORE_FULL`.
