# AI News Digest

A Go-based daily AI news digest that fetches recent articles from curated sources, ranks the strongest candidates, asks OpenAI to produce a concise digest, and sends the result to Telegram.

## Current Scope
- Delivery: Telegram only
- Scheduler: GitHub Actions
- Source mode: RSS only in the first version
- Sources included:
  - OpenAI
  - Anthropic
  - TechCrunch AI
  - Google DeepMind
  - Meta AI
  - The Verge AI
- No database
- No historical storage
- No cross-day deduplication

## How It Works
The CLI runs a single batch pipeline:

`fetch -> normalize -> recent filter -> dedupe -> score -> LLM select/summarize -> telegram`

Behavior highlights:
- Only articles from the last 24 hours are considered.
- Official sources are ranked above media sources.
- Duplicate coverage is collapsed within the current run.
- If OpenAI fails, the app falls back to rule-ranked items.
- If there are fewer than 3 strong items, it sends only 1 or 2.

## Project Structure
```text
cmd/ai-news-digest/        CLI entrypoint
configs/sources.yaml       Source configuration
internal/config/           Config loading
internal/model/            Shared data models
internal/source/           RSS providers and source collection
internal/pipeline/         Filtering, dedupe, scoring, fallback
internal/llm/              OpenAI selection and summary generation
internal/deliver/          Telegram sender
internal/format/           Telegram message formatting
.github/workflows/         GitHub Actions workflow
```

## Requirements
- Go 1.22+
- OpenAI API key
- Telegram bot token
- Telegram chat ID

## Environment Variables
Copy `.env.example` into your own environment management setup.

Required:
- `OPENAI_API_KEY`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

Optional:
- `OPENAI_MODEL`
  - Default: `gpt-4.1-mini`

## Local Development
Install dependencies and run tests:

```bash
go mod tidy
go test ./...
```

Run the digest locally:

```bash
OPENAI_API_KEY=... \
TELEGRAM_BOT_TOKEN=... \
TELEGRAM_CHAT_ID=... \
go run ./cmd/ai-news-digest
```

## GitHub Actions Setup
The workflow is defined in `.github/workflows/daily-digest.yml`.

It supports:
- scheduled execution at `01:00 UTC` (`09:00 Asia/Taipei`)
- manual execution via `workflow_dispatch`

Add these repository secrets before running the workflow:
- `OPENAI_API_KEY`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

## Source Configuration
Current sources are defined in `configs/sources.yaml`.

Each source includes:
- `name`
- `type` (`official`, `media`, `social`)
- `mode` (`rss`)
- `url`
- `enabled`
- `include_keywords` (optional)

## Testing
Current automated coverage focuses on the parts most likely to break:
- pipeline dedupe and fallback logic
- Telegram message formatting
- RSS parsing with fixtures

Run:

```bash
go test ./...
```

## Known Limitations
- Source coverage is still intentionally selective.
- HTML parsers are not implemented yet.
- Social sources are not implemented yet.
- Telegram message length splitting is not implemented yet.
- There is no persistent storage, so cross-day duplicates are still possible.

## Next Suggested Improvements
- Add more sources beyond the current six feeds
- Add HTML parser support for sources without stable RSS feeds
- Improve event clustering beyond normalized-title matching
- Add message length guards for Telegram
