# Отчёт: Этап 5 (задачи 5.1–5.5) — HTTP API

- **Дата:** 2026-10-29
- **Агент:** backend
- **Этап и задачи плана:** Этап 5, задачи 5.1–5.5
- **Предыдущий отчёт:** `docs/reports/2026-10-29_01_backend_etap-4-benchmark-modul.md`

## 1. Краткое резюме

Реализован HTTP-слой приложения: сервер на стандартном `net/http` (Go 1.22 ServeMux с method+wildcard
паттернами), загрузка и генерация датасетов, асинхронное построение HNSW-индекса с прогрессом через SSE,
вставка одиночных векторов и k-NN поиск в режимах `hnsw` / `bruteforce` (с трассировкой по слоям).
Добавлен отдельный пакет `internal/indexmanager`, который владеет текущим индексом, его состоянием,
отображением внешних id, точным bruteforce-двойником и подписчиками SSE; пакет `internal/api` только
транслирует HTTP в вызовы менеджера. Контракт `docs/API.md` расширен только там, где реализация
уточняет поведение (health, коды ошибок, значения по умолчанию, формат id и расстояний).

## 2. Что сделано

- **5.1** — пакет `backend/internal/api`: `Server`, `Routes()`/`Handler()`, маршруты `/api/v1/...`,
  CORS для `http://localhost:5173` (preflight `OPTIONS` → 204), единый конверт ошибок
  `{"error":{"code","message"}}`, обработка неизвестного маршрута (`NOT_FOUND`) и неверного метода
  (`METHOD_NOT_ALLOWED`), panic-recovery, `GET /api/v1/health`.
  `backend/cmd/server/main.go`: `PORT` (по умолчанию 8080), `signal.NotifyContext` (SIGINT/SIGTERM),
  `http.Server.Shutdown` с таймаутом 10 с, логи старта/остановки.
- **5.2** — `POST /api/v1/datasets`: multipart-загрузка `.csv`/`.json` (лимит 256 MiB, неподдерживаемое
  расширение — 415) и генерация (`action:"generate"`, распределения `uniform|gaussian|clusters`,
  лимит `n*dim <= 50_000_000`); `GET /datasets` (список), `GET /datasets/{id}` (метаданные, preview,
  stats); ошибки `DATASET_NOT_FOUND`, `INVALID_REQUEST`.
- **5.3** — `internal/indexmanager`:
  - `POST /index` → 202 `{indexId,status:"building"}`, сборка идёт в горутине, прогресс обновляется
    после каждой вставки (публикация примерно каждые 1 %);
  - `GET /index/status` → `idle|building|ready|error`, `progress`, `elapsedMs`, `indexId`, `error`;
  - `GET /index/events` (SSE) → события `progress`, `done`, `error`, снимок текущего состояния при
    подключении, heartbeat-комментарии, закрытие после терминального события или отключения клиента;
  - `GET /index/stats` → `nodes`, `maxLevel`, `entryPoint`, `nodesPerLevel`,
    `degreeDistribution {min,max,mean,p50}`, `metric`, `M`, `efConstruction` (+ `indexId`, `datasetId`).
  - Повторная сборка во время сборки отклоняется с 409 `INDEX_NOT_READY`; пересборка готового индекса
    разрешена и заменяет предыдущий индекс (во время сборки поиск/вставка/статистика недоступны).
  - `GET /index/stats`: `INDEX_NOT_FOUND` до первой сборки, `INDEX_NOT_READY` во время сборки,
    `BUILD_FAILED` при ошибке.
- **5.4** — `POST /vectors`: вставка одного вектора в готовый индекс, `vectorId` опционален
  (сервер выдаёт `ext_<n>`), ответ 201 `{id,vectorId}`; проверяются готовность, размерность, NaN/Inf и
  дубликаты; bruteforce-двойник и таблицы id обновляются атомарно с вставкой в HNSW.
- **5.5** — `POST /search`: режимы `hnsw` (с `ef = max(efSearch,k)`) и `bruteforce`; запрос по `vector`
  или `vectorId` (строка или число), `trace` только для HNSW, ответ `{results:[{id,vectorId,distance,vector}],found,trace?}`.
  Для L2 в ответе применяется `sqrt`, cosine отдаётся как `1 - similarity`, inner product — как
  отрицательное скалярное произведение (меньше = ближе). Ошибки `K_EXCEEDS_N`, `INVALID_DIMENSION`,
  `INDEX_NOT_FOUND`, `INDEX_NOT_READY`, `INVALID_REQUEST`, `SEARCH_FAILED`.

Задачи 5.6–5.10 не реализовывались; 5.6 оставлена незакрытой (используется существующий
`SearchWithTrace`, но соответствие контракту проверено только на уровне формы ответа).

## 3. Изменённые и добавленные файлы

| Файл | Что изменено |
|---|---|
| `backend/internal/indexmanager/manager.go` | новый пакет: состояние сборки, async build, SSE-подписки, id-маппинг, вставка, поиск, статистика |
| `backend/internal/indexmanager/manager_test.go` | тесты сборки/поиска/вставки, повторной сборки, отмены и ошибок «не готов» |
| `backend/internal/api/server.go` | новый пакет: `Server`, маршруты, CORS, `writeJSON`/`writeError`, `FlexID`, маппинг ошибок |
| `backend/internal/api/datasets.go` | `POST/GET /datasets`, multipart и generate |
| `backend/internal/api/index.go` | `POST /index`, `GET /index/status`, `GET /index/stats` |
| `backend/internal/api/sse.go` | `GET /index/events` (SSE) |
| `backend/internal/api/vectors.go` | `POST /vectors` |
| `backend/internal/api/search.go` | `POST /search`, преобразование расстояний и трассировки |
| `backend/internal/api/api_test.go` | httptest-тесты всех эндпоинтов, CORS, SSE, ошибок и валидации |
| `backend/cmd/server/main.go` | точка входа, `PORT`, сигналы, graceful shutdown, `run()` для теста |
| `backend/cmd/server/main_test.go` | тест запуска и остановки по отмене контекста |
| `docs/API.md` | health, новые коды ошибок, значения по умолчанию, семантика id/расстояний, SSE |
| `docs/PLAN.md` | отметки 5.1–5.5, статус, решения, отчёт и change log |

## 4. Ключевые решения

- **Решение:** отдельный `internal/indexmanager` вместо хранения состояния в HTTP-обработчиках.
  **Альтернативы:** manager внутри `internal/api`; глобальные переменные.
  **Причина:** вся конкурентная логика (блокировки, сборка, подписчики, id-маппинг) изолирована от HTTP;
  `api` не импортирует `hnsw`/`bruteforce` напрямую, кроме типов метрик и трассировки. Порядок блокировок
  единый: `indexmanager` → `hnsw.Index`, что исключает дедлоки.
- **Решение:** сборка идёт в отдельной горутине; новый индекс публикуется только после полной вставки.
  **Альтернативы:** синхронная сборка в обработчике; потоковая публикация частично готового индекса.
  **Причина:** контракт требует немедленный 202 и SSE-прогресс; частично готовый индекс нельзя безопасно
  искать без дополнительных блокировок.
- **Решение:** пересборка сбрасывает предыдущий индекс; повторная сборка во время сборки → 409
  `INDEX_NOT_READY`.
  **Альтернативы:** держать старый индекс доступным до готовности нового; очередь сборок.
  **Причина:** простая и предсказуемая семантика; состояние гонки «какой индекс искать» исключено.
- **Решение:** внешние id датасетных векторов — десятичные строки `"0".."N-1"`, серверные id вставок —
  `"ext_<n>"`; `FlexID` принимает и строку, и число в `/search`.
  **Альтернативы:** только строки; только числа; отдельная таблица в ядре.
  **Причина:** пример контракта для `/search` использует число (`"vectorId": 42`), а `/vectors` — строку
  (`"ext_123"`); `FlexID` поддерживает оба варианта без изменения ядра HNSW.
- **Решение:** bruteforce-индекс перестраивается при каждой вставке.
  **Альтернативы:** держать только исходный датасет и документировать расхождение.
  **Причина:** результаты `hnsw` и `bruteforce` остаются сопоставимыми после `POST /vectors`; цена —
  O(N) на одну вставку, что приемлемо для учебного API и малых N.
- **Решение:** `degreeDistribution` считается по `hnsw.Stats().DegreeHistogram`, то есть по всем парам
  `(узел, уровень)`.
  **Альтернативы:** менять ядро и считать только уровень 0.
  **Причина:** не ломаем API ядра; для типичных `M` уровень 0 доминирует, а семантика явно описана в
  `docs/API.md`.
- **Решение:** не добавлять сторонние зависимости.
  **Причина:** хватило стандартной библиотеки (`net/http`, `mime/multipart`, `encoding/json`, `bufio`).

## 5. Проверка и результаты

### Форматирование, vet, сборка

```
$ cd backend && gofmt -l cmd/server internal/api internal/indexmanager
(пусто)

$ go vet ./...
(пусто, код 0)

$ go build ./...
(пусто, код 0)
```

`gofmt -l .` по всему `backend/` показывает старые файлы этапов 1–4 (переформатирование под текущую
версию `gofmt` не выполнялось, чтобы не смешивать изменения); все затронутые в этой задаче файлы
отформатированы.

### Тесты

```
$ go test ./... -count=1
ok  hnsw-coursework/cmd/server            0.123s
ok  hnsw-coursework/internal/api         0.101s
ok  hnsw-coursework/internal/benchmark    2.081s
ok  hnsw-coursework/internal/bruteforce   0.027s
ok  hnsw-coursework/internal/dataset      0.043s
ok  hnsw-coursework/internal/hnsw        14.728s
ok  hnsw-coursework/internal/indexmanager 0.085s
ok  hnsw-coursework/internal/projection   0.014s

$ go test -race ./... -count=1
ok  hnsw-coursework/cmd/server            1.145s
ok  hnsw-coursework/internal/api         1.284s
ok  hnsw-coursework/internal/benchmark    7.117s
ok  hnsw-coursework/internal/bruteforce   1.126s
ok  hnsw-coursework/internal/dataset      1.149s
ok  hnsw-coursework/internal/hnsw       231.499s
ok  hnsw-coursework/internal/indexmanager 1.366s
ok  hnsw-coursework/internal/projection   1.048s
```

Покрытые сценарии (`internal/api/api_test.go`, `internal/indexmanager/manager_test.go`,
`cmd/server/main_test.go`):

- health, единый конверт ошибок, 404 на неизвестный маршрут, 405 на неверный метод;
- CORS preflight и обычный запрос с `Origin`, отсутствие CORS-заголовка для чужого origin;
- генерация/список/получение датасета, CSV/JSON multipart, неподдерживаемый тип файла, битый CSV;
- сборка индекса, polling статуса до `ready`, статистика, ошибки неизвестного датасета и метрики,
  пересборка;
- SSE: начальный комментарий `: idle`, события `progress` и `done`, немедленный `done` для готового
  индекса, остановка обработчика при отключении клиента и при `Manager.Close()`;
- вставка вектора с явным и автоматическим id, дубликат, неверная размерность, поиск по `vectorId`
  (строка и число), bruteforce видит вставленный вектор;
- поиск `hnsw`/`bruteforce`, сортировка по расстоянию, `trace` только для HNSW, `K_EXCEEDS_N`,
  неверный режим, одновременная передача `vector` и `vectorId`, отсутствие запроса, неверная размерность;
- запуск `run()` на свободном порту и остановка по отмене контекста (аналог SIGINT/SIGTERM).

### Ручной smoke-тест

Сервер собран и запущен на свободном порту 18081; запросы выполнялись `curl`, JSON разбирался Python.
Фактический вывод (сокращённо):

```
== health ==
{"status":"ok"}
== generate ==
{"id": "ds_000001", "n": 200, "dim": 8}
== list ==
[{'id': 'ds_000001', 'n': 200, 'dim': 8}]
== build ==
{"indexId":"idx_fd33405c","status":"building"}
== status ==
{"status":"ready","progress":1,"elapsedMs":3,"indexId":"idx_fd33405c","error":null}
== stats ==
{"nodes": 200, "maxLevel": 3, "entryPoint": 117, "nodesPerLevel": [200, 22, 2, 1],
 "degreeDistribution": {"min": 0, "max": 16, "mean": 8.16, "p50": 8},
 "metric": "l2", "M": 8, "efConstruction": 50, "indexId": "idx_fd33405c"}
== search hnsw (trace) ==
{"found": 5, "results": [[0, "0", 0], [138, "138", 0.530393], [19, "19", 0.599794],
 [134, "134", 0.636852], [11, "11", 0.655052]],
 "traceLayers": [[3, 1, 117], [2, 2, 100], [1, 3, 149], [0, 21, 0]]}
== search bruteforce ==
{"found": 5, "results": [[0, "0", 0], [138, "138", 0.530393], [19, "19", 0.599794],
 [134, "134", 0.636852], [11, "11", 0.655052]]}
== insert vector ==
{"id":200,"vectorId":"custom"}
== search by vectorId ==
{"found": 3, "results": [[200, "custom", 0], [0, "0", 0.5], [170, "170", 0.586716]]}
== stats after insert ==
{"nodes": 201, "degreeDistribution": {"min": 0, "max": 16, "mean": 8.1674, "p50": 8}}
== shutting down ==
```

На этом прогоне HNSW и bruteforce вернули одинаковый top-5 (recall@5 = 1.0 для запроса, совпадающего
с вектором датасета); `trace` содержит слои 3→0, на уровне 0 посещён 21 узел. После вставки
`custom` узел `200` найден первым с расстоянием 0, а bruteforce также видит его.

**Что не проверено:**

- ручная остановка через `kill -TERM` в Git Bash на Windows не доставляет SIGTERM в Go-процесс (процесс
  завершается, лог `api: stopped` не появляется); корректность `Shutdown` проверена тестом
  `TestRunStartsAndShutsDown` через отмену контекста, а не реальным сигналом ОС;
- SSE под нагрузкой (1M векторов, длительная сборка, сотни клиентов) не тестировался;
- лимит 256 MiB и `n*dim <= 50M` проверены по коду, без реальной загрузки файла такого размера;
- эндпоинты `GET /graph` (5.7) и `POST /benchmark` (5.8) ещё не реализованы.

## 6. Проблемы и ограничения

- При пересборке старый индекс сразу становится недоступен: поиск во время сборки возвращает
  `INDEX_NOT_READY`. Это осознанный выбор, чтобы не смешивать результаты старого индекса с новым
  датасетом.
- `POST /vectors` перестраивает bruteforce-индекс за O(N); для больших N это заметно. В будущем можно
  добавить инкрементальный bruteforce или отдельный режим «только HNSW».
- `degreeDistribution` включает верхние уровни; если для визуализации нужна гистограмма строго уровня 0,
  потребуется добавить в ядро метод `DegreesAtLevel(0)` (не делалось, чтобы не расширять ядро в рамках 5.5).
- `POST /datasets` читает файл целиком в память (так устроены существующие загрузчики CSV/JSON); лимит
  256 MiB ограничивает худший случай, но не отменяет двойной расход памяти при JSON.
- SSE-события прогресса throttled примерно до 100 на сборку; при очень маленьких N событий меньше, что
  нормально для UI.
- При ошибке сборки `vectors`/`Len()` остаются от нового датасета, но `idx == nil`; все публичные
  операции сначала проверяют статус, поэтому частичный индекс наружу не виден.

## 7. Изменения контракта API / влияние на другую часть

Контракт в целом соблюдён; дополнения зафиксированы в `docs/API.md`:

- добавлен `GET /api/v1/health` → `200 {"status":"ok"}`;
- добавлены коды ошибок `FILE_TOO_LARGE` (413), `UNSUPPORTED_FILE_TYPE` (415), `NOT_FOUND` (404),
  `METHOD_NOT_ALLOWED` (405), `INTERNAL_ERROR` (500);
- значения по умолчанию `POST /index`: `M=16`, `efConstruction=200`, `metric=l2`, `heuristic=true`, `seed=0`;
- `vectorId` в `/search` принимается строкой или числом; id датасетных векторов — `"0".."N-1"`,
  серверные id вставок — `"ext_<n>"`;
- `trace` возвращается только для HNSW;
- `degreeDistribution` считается по всем `(узел, уровень)`;
- при повторной сборке предыдущий индекс заменяется, во время сборки — `INDEX_NOT_READY`;
- `/index/stats` может содержать `indexId` и `datasetId`.

Для фронтенда важно: SSE теперь можно подключать и до старта сборки (придёт `: idle`, затем прогресс);
`done` закрывает поток; при готовом индексе поток сразу отдаёт `done` и закрывается.

## 8. Материал для курсовой

| Глава | Что можно использовать |
|---|---|
| Глава 1 (структура, параметры) | описание асинхронной сборки индекса, параметров `M`/`efConstruction`/`efSearch`, отображения внешних id, разделения «ядро HNSW — менеджер — HTTP»; пример статистики (`nodesPerLevel`, степень вершин) |
| Глава 2 (визуализация) | формат SSE-прогресса сборки, формат `trace` (`layers[].level/visited/closest`), внешние id и связь с будущим `GET /graph`; скриншоты прогресса и поиска можно делать уже сейчас |
| Глава 3 (сравнение с перебором) | единый `/search` с режимами `hnsw`/`bruteforce`, одинаковый формат результатов и расстояний; smoke-пример совпадения top-5; база для будущего `/benchmark` |

## 9. Следующие шаги

1. 5.6 — проверить и при необходимости доработать `SearchWithTrace` так, чтобы `trace` полностью
   соответствовал контракту (сейчас форма ответа совпадает, но 5.6 оставлена незакрытой).
2. 5.7 — `GET /graph` (`level`, `limit`, `around`, PCA-координаты) на основе `projection.Cache` и
   данных индекса.
3. 5.8 — `POST /benchmark` с использованием готового `internal/benchmark`.
4. 5.9–5.10 — сохранение/загрузка индекса (опционально) и финальная синхронизация `docs/API.md`.
