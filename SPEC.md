# study-blocks

A single-binary Go + Svelte app to track daily study minutes (via Telegram or UI) and visualize the last 30 days.

## Current Product Behavior

- Rolling 30-day chart (today on the right)
- Grouped bars by subject (side-by-side, not stacked)
- Topic filter by clicking subject name in legend
- Click **Study Blocks** title to clear filter (show all subjects)
- Click a chart day/bar to open a modal and set minutes for a chosen topic/date
- Add topics from UI (`add` button in legend row)
- Subject colors are editable and persisted in DB
- Auto-refresh every **20 minutes**

---

## Architecture

```
study-blocks/
├── cmd/server/main.go
├── internal/
│   ├── app/app.go
│   ├── config/config.go
│   ├── handler/
│   │   ├── api.go
│   │   └── frontend.go
│   ├── model/
│   │   ├── entry.go
│   │   └── subject.go
│   ├── service/
│   │   ├── ingest.go
│   │   └── telegram.go
│   └── store/store.go
├── frontend/
│   └── src/App.svelte
└── web/dist/ (embedded frontend)
```

- Go backend + BoltDB persistence
- Svelte + Chart.js frontend, embedded in binary
- Telegram bot integration is optional

---

## Data Model

### Subject

```go
type Subject struct {
    Name  string
    Color string
}
```

- Subjects are persisted in BoltDB metadata (`subjects` key)
- Can be added at runtime via API/UI
- Color changes are persisted in DB

### Entry

```go
type Entry struct {
    ID        string
    Timestamp int64
    Date      string // YYYY-MM-DD
    Subject   string
    Minutes   int
}
```

- Bolt key: `YYYY-MM-DD:timestamp:id`
- Bucket: `entries`

---

## Subject Bootstrap Rules

`SUBJECTS` env var is now treated as **seed/default subjects**, not strict runtime source of truth.

Startup behavior:

1. If DB already has subjects → use DB subjects.
2. If DB has none → seed from `SUBJECTS`.
3. If DB has subjects and `SUBJECTS` contains additional new subjects → append missing ones.
4. If both are empty → startup error.

This prevents redeploys from wiping runtime-added subjects.

---

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | Health check |
| GET | `/api/subjects` | List subjects |
| POST | `/api/subjects` | Add subject (`{"name":"writing"}`) |
| PATCH | `/api/subjects/{name}` | Update subject color (`{"color":"#aabbcc"}`) |
| GET | `/api/entries?from=YYYY-MM-DD&to=YYYY-MM-DD` | List entries |
| POST | `/api/entries` | Create entry (`date`, `subject`, `minutes`) |
| DELETE | `/api/entries/{id}` | Delete entry |

---

## Frontend Behavior

### Chart

- 30-day grouped bar chart
- Subjects are side-by-side per day
- If a subject has no value for a day, dataset uses `null` (not `0`) so single bars center naturally
- Fixed narrow bar thickness, even when only one topic exists that day
- Tooltip title: full weekday + day + month (localized)

### Topic interactions

- Click subject name in legend to filter chart to one topic
- Click again to toggle off
- Click title (**Study Blocks**) to return to all topics

### Editing study minutes

- Click chart day/bar to open modal
- Modal lets user select topic + input minutes
- Save logic is “set total for topic/date”:
  - delete existing entries for that topic/date
  - create one new entry if minutes > 0

### Subject management

- `add` button in legend row opens topic-name prompt
- Topic color swatch opens color picker
- Color changes call backend and persist in DB (no localStorage override)

### Refresh

- Polls entries/subjects every 20 minutes
- Midnight refresh still updates the rolling window
- Frontend avoids unnecessary chart churn by only replacing state when fetched payload changed

---

## Telegram Input

Supported commands/messages:

- `<minutes> <subject>` or `<subject> <minutes>`
- `subjects`
- `today`
- `undo`

Telegram config:

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_ALLOWED_CHAT_ID`

---

## Configuration

| Env Var | Default | Notes |
|---|---|---|
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `BOLTDB_PATH` | `data/study-blocks.db` | DB path |
| `SUBJECTS` | `""` | Seed/default subject list; required only when DB has no subjects |
| `TELEGRAM_BOT_TOKEN` | `""` | Optional |
| `TELEGRAM_ALLOWED_CHAT_ID` | `0` | 0 = allow all |
| `ENABLE_LOCAL_TEST_ROUTES` | `false` | Test-only clear route |

---

## TODO: Daily “No Input” Telegram Reminder

Add a daily reminder that texts the user when no study input was logged that day.

### Proposed implementation

1. Add a reminder loop in `internal/service/telegram.go`
2. At configured local time, run `ingest.ListEntries(today, today)`
3. If zero entries, send Telegram reminder via existing `sendMessage`
4. Persist `last_reminder_date` in Bolt metadata to avoid duplicates
5. Add timezone support (e.g. `APP_TZ`) for correct day boundaries
6. Add config toggles:
   - `DAILY_REMINDER_ENABLED`
   - `DAILY_REMINDER_HOUR`
   - `DAILY_REMINDER_MINUTE` (optional)

### Open decisions

- Reminder only on zero entries vs threshold
- Behavior if app is down at reminder time
- Behavior when `TELEGRAM_ALLOWED_CHAT_ID=0`
