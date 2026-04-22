# study-blocks

A single-binary Go + Svelte app that tracks daily study time via Telegram and displays a rolling 30-day stacked bar chart. Follows the same architecture as `calorie-counter`.

## Concept

Inspired by a college study sheet: a simple grid where you color in 20-minute blocks each day to see that you're making continuous progress. The goal is not to hit a target — it's to look at 30 days and see that you showed up.

---

## Architecture

Identical to `calorie-counter`:

```
study-blocks/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── app/app.go                   # Wiring & lifecycle
│   ├── config/config.go             # Env var config
│   ├── handler/
│   │   ├── api.go                   # HTTP API handlers
│   │   └── frontend.go              # Embedded static file serving
│   ├── model/
│   │   ├── entry.go                 # Study entry model
│   │   └── subject.go               # Subject definitions
│   ├── service/
│   │   ├── ingest.go                # Message parsing & storage
│   │   └── telegram.go              # Telegram message handler
│   ├── store/store.go               # BoltDB persistence
│   └── telegram/
│       ├── poller.go                # Long-polling loop
│       └── types.go                 # Telegram API types
├── frontend/
│   ├── src/
│   │   ├── App.svelte               # Main component (chart + legend)
│   │   └── main.js                  # Entry point
│   ├── vite.config.js
│   └── package.json
├── web/
│   ├── assets.go                    # go:embed
│   └── dist/                        # Built frontend
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

- **Go 1.25**, **Svelte 5**, **Vite**, **BoltDB**, **Chart.js** via `svelte-chartjs`
- No LLM dependency (input is structured, not natural language)
- Frontend compiled and embedded into the Go binary via `embed.FS`

---

## Data Model

### Subject (configured at startup)

Subjects are defined in an environment variable as a comma-separated list:

```
SUBJECTS=math,physics,cs,writing
```

Each subject gets a deterministic color assigned by its position in the list. The subject list is the source of truth — entries referencing unknown subjects are rejected.

### Entry

```go
type Entry struct {
    ID        string // UUID
    Timestamp int64  // Unix timestamp of when the entry was logged
    Date      string // YYYY-MM-DD (the day the study happened)
    Subject   string // Must match a configured subject
    Blocks    int    // Number of 20-minute blocks
}
```

**BoltDB key format:** `YYYY-MM-DD:timestamp:id` (same pattern as calorie-counter, enables prefix scan by date)

**BoltDB buckets:**
- `entries` — study entry records
- `metadata` — telegram offset, failure counts

---

## Telegram Bot Input

### Input Format

```
<blocks> <subject>
```

Examples:
- `2 math` — log 2 blocks (40 min) of math for today
- `3 physics` — log 3 blocks (60 min) of physics for today

### Parsing Rules

1. Split message on first space: `blocks` (integer) and `subject` (string)
2. Validate `blocks` is a positive integer
3. Validate `subject` matches a configured subject (case-insensitive, stored lowercase)
4. Date is always **today** (derived from timestamp at time of message processing)
5. On success: save entry, reply with confirmation (e.g. `Logged 2 blocks of math (40 min)`)
6. On failure: reply with error message explaining what went wrong

### Special Commands

- `subjects` — reply with the list of configured subjects
- `today` — reply with today's entries and total blocks per subject
- `undo` — delete the most recent entry for today

### Telegram Config

Same pattern as calorie-counter:

| Env Var | Purpose |
|---------|---------|
| `TELEGRAM_BOT_TOKEN` | Bot API token |
| `TELEGRAM_ALLOWED_CHAT_ID` | Restrict to one chat (0 = allow all) |

Telegram is optional — app runs without it if token is empty.

---

## API

All routes registered on a single `http.ServeMux`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/subjects` | Return configured subjects with their colors |
| GET | `/api/entries?from=YYYY-MM-DD&to=YYYY-MM-DD` | Return entries for a date range |
| POST | `/api/entries` | Create an entry (body: `{"date": "YYYY-MM-DD", "subject": "math", "blocks": 2}`) |
| DELETE | `/api/entries/{id}` | Delete an entry by ID |
| GET | `/` | Serve embedded Svelte frontend |

### GET /api/subjects

Response:
```json
[
  { "name": "math", "color": "#7c6ef0" },
  { "name": "physics", "color": "#f07c6e" },
  { "name": "cs", "color": "#6ef0a2" },
  { "name": "writing", "color": "#f0d86e" }
]
```

Colors are deterministic based on position in the `SUBJECTS` list using a fixed palette or HSL rotation.

### GET /api/entries?from=2026-03-24&to=2026-04-22

Response:
```json
[
  {
    "id": "abc123",
    "timestamp": 1745352000,
    "date": "2026-04-22",
    "subject": "math",
    "blocks": 2
  }
]
```

### POST /api/entries

Request:
```json
{
  "date": "2026-04-22",
  "subject": "math",
  "blocks": 2
}
```

Response: `201 Created` with the created entry.

---

## Frontend (Svelte)

### Single View: Rolling 30-Day Stacked Bar Chart

The page shows one thing: a stacked bar chart covering exactly the last 30 days (today and the 29 days before it).

```
Blocks
  8 |
  7 |                         ██
  6 |              ██         ██
  5 |    ██        ██    ██   ██
  4 |    ██   ██   ██    ██   ██
  3 |    ██   ██   ██    ██   ██
  2 | █  ██   ██   ██    ██   ██
  1 | █  ██   ██   ██    ██   ██
    +--+--+--+--+--+--+--+--+--+--
     3/24 3/25 3/26 ...      4/22

█ = colored per subject
```

**Chart behavior:**

- **X-axis:** 30 columns, one per day. Labels show short date (e.g., `Mar 24`, `Apr 1`). Today is always the rightmost column.
- **Y-axis:** Count of 20-minute blocks. Starts at 0. Auto-scales to fit the tallest day.
- **Bars:** Each day's bar is composed of stacked colored segments, one per subject. Subjects always stack in the same order (matching the configured list order) for visual consistency.
- **Colors:** Each subject has a fixed color from `/api/subjects`.
- **Empty days:** Show as a gap (no bar) — this is intentional. The visual gap is the motivation to not break the streak.
- **Today highlight:** Today's column has a subtle visual distinction (e.g., slightly brighter border or background highlight) so you can see where "now" is.

**Legend:**

Below the chart, a simple legend showing each subject with its color swatch and name.

**On load:**

1. Fetch `/api/subjects` to get subject names and colors
2. Fetch `/api/entries?from=<30 days ago>&to=<today>` to get all entries for the window
3. Group entries by date, then by subject within each date
4. Sum blocks per subject per day
5. Build chart datasets (one dataset per subject, each with 30 data points)
6. Render

**Auto-refresh:** Poll `/api/entries` every 60 seconds to pick up new Telegram entries without manual refresh.

### Styling

- Dark theme matching calorie-counter (background `#020617`, text `#f8fafc`)
- Glassmorphism card containing the chart
- Clean, minimal — the chart is the entire UI
- Responsive: chart fills available width, reasonable min-height on mobile

---

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `BOLTDB_PATH` | `data/study-blocks.db` | Database file path |
| `SUBJECTS` | (required) | Comma-separated subject list |
| `TELEGRAM_BOT_TOKEN` | `""` | Telegram bot token (optional) |
| `TELEGRAM_ALLOWED_CHAT_ID` | `0` | Restrict to one chat |

No LLM config needed — input parsing is deterministic.

---

## Build & Deploy

Same Makefile targets and Docker multi-stage build as calorie-counter:

- `make run` — build frontend, run server
- `make build` — compile binary
- `make docker-build` — multi-stage Docker image
- `make up` / `make down` — docker-compose

Docker Compose mounts a persistent volume for the BoltDB file.

---

## What This App Does NOT Do

- No goal/target lines — the point is just to see continuity
- No analytics, averages, or streaks counter — the chart speaks for itself
- No web-based input — Telegram only
- No LLM — input is structured (`<blocks> <subject>`)
- No month boundaries — always a rolling 30-day window ending today
- No user accounts — single user, single Telegram chat
