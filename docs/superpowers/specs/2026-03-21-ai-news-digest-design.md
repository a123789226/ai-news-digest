# AI News Digest Design

## Overview
Build a Go-based daily AI news digest tool that runs once per day via GitHub Actions at 07:12 Asia/Taipei. The tool fetches AI-related news from a curated whitelist of official and mainstream media sources, filters to the last 24 hours, deduplicates overlapping coverage, ranks candidates with deterministic rules, then uses OpenAI to select and summarize the most important 1 to 3 items. The final digest is delivered to Telegram only.

The system is intentionally stateless. It does not store historical data, does not run as a long-lived service, and does not attempt cross-day deduplication. If a given day has fewer than three high-quality items, it sends only one or two rather than padding with low-value content.

## Goals
- Deliver a concise daily AI digest to Telegram at 07:12 Asia/Taipei.
- Prioritize official sources, then mainstream media, with social signals used only as supporting input.
- Keep the implementation simple and maintainable: Go CLI, GitHub Actions scheduler, no database.
- Preserve output quality by using deterministic filtering and scoring before any LLM involvement.
- Gracefully degrade when some sources or the OpenAI API fail.

## Non-Goals
- No Discord delivery.
- No Twitter/X integration.
- No Reddit integration.
- No web UI or admin panel.
- No persistent storage or historical tracking.
- No cross-day deduplication.
- No long-running server or embedded scheduler.

## Deployment Model
The application is a single-run Go CLI executed by GitHub Actions.

Workflow characteristics:
- Scheduled daily at `23:12 UTC`, equivalent to `07:12 Asia/Taipei`.
- Supports `workflow_dispatch` for manual runs.
- Uses GitHub Secrets for runtime configuration.
- Writes operational logs to GitHub Actions output.

Required secrets:
- `OPENAI_API_KEY`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

## High-Level Architecture
The system is organized as a linear batch pipeline:

`fetch -> normalize -> recent filter -> dedupe -> score -> LLM select/summarize -> telegram`

Module boundaries:
- `source`: adapters for RSS/Atom feeds and selected HTML-backed sources.
- `pipeline`: filtering, deduplication, scoring, candidate preparation.
- `llm`: OpenAI integration for final selection and summary generation.
- `deliver`: Telegram sender.
- `config`: source definitions, weights, scoring knobs, environment config.

Design rule:
- The LLM does not discover news. It only evaluates already-collected candidates and produces final summaries.

## Source Strategy
### Source Tiers
Official sources:
- OpenAI
- Anthropic
- Google DeepMind / Google Blog AI-related sections
- Meta AI
- Microsoft AI / Research
- NVIDIA Blog

Mainstream media:
- TechCrunch AI
- The Verge AI
- MIT Technology Review AI
- VentureBeat AI
- Reuters Technology

Social signal:
- Hacker News

### Source Rules
- Prefer RSS/Atom whenever available.
- Use HTML parsing only for a small number of high-value sources that lack a usable feed.
- Treat Hacker News as a signal source only. It cannot enter the final digest unless corroborated by an official or mainstream media source.
- Exclude X/Twitter, Reddit, login-gated sources, and fragile anti-bot targets from the first version.

### Source Adapter Contract
Each source adapter returns normalized `Article` values.

Proposed model:

```go
type Article struct {
    Source       string
    SourceType   string // official, media, social
    Title        string
    URL          string
    PublishedAt  time.Time
    SummaryRaw   string
    ContentRaw   string
}
```

## Data Flow
### 1. Fetch
Collect candidate entries from all configured sources.

Behavior:
- Source failures are isolated and do not abort the whole run.
- Each adapter returns best-effort metadata: title, URL, timestamp, and summary or snippet when available.

### 2. Normalize
Convert all source output into the common `Article` structure.

Normalization responsibilities:
- Normalize source labels.
- Normalize source type.
- Parse and validate publication timestamps.
- Trim noisy whitespace and HTML leftovers in summary fields.

### 3. Recent Filter
Keep only items published within the most recent 24 hours.

Rules:
- If a source does not provide a trustworthy publication time, either discard the item or apply a severe downgrade depending on adapter reliability.
- The first version should prefer dropping uncertain items over guessing.

### 4. Deduplication
Deduplicate only within the current run.

Deduplication steps:
- Exact URL dedupe.
- Normalized-title similarity check.
- Event grouping for overlapping coverage.

Event merge rule:
- When official and media articles describe the same event, keep the official article as the primary candidate.
- Keep media coverage only as supporting evidence for scoring, not as a separate final digest item.

### 5. Rule-Based Scoring
Apply deterministic scoring before calling the LLM.

Signals include:
- Source type priority: official > media > social.
- Recency within the 24-hour window.
- High-value keywords such as `release`, `launch`, `model`, `api`, `funding`, `policy`, `research`, `open source`.
- Multi-source corroboration.
- Penalties for opinion pieces, tutorials, and low-information content.

Scoring goals:
- Promote significant releases and industry-relevant events.
- Reduce duplicate story variants.
- Ensure only strong candidates reach the LLM stage.

### 6. LLM Selection and Summary
Send only the highest-ranked candidates to OpenAI.

LLM responsibilities:
- Select the most important 1 to 3 items.
- Generate structured output for each selected item.

Required fields per item:
- Original English title.
- Chinese summary.
- `Why it matters` explanation.
- Source name.
- URL.

Selection guidance:
- Prefer high-impact product, model, policy, research, or business developments.
- Avoid choosing multiple near-duplicate events.
- It is acceptable to return fewer than three items.

Structured output should be JSON to make parsing deterministic.

### 7. Delivery
Format the selected items into a Telegram-friendly message and send once per run.

Message requirements:
- Clear separation between items.
- Preserve the original English title.
- Provide Chinese summary and `Why it matters`.
- Include source and URL.
- Prefer plain text or very conservative formatting to reduce Telegram formatting failures.

## Failure Handling and Fallbacks
### Source Failures
- A failed source adapter logs an error and returns no items.
- The rest of the pipeline continues.

### OpenAI Failures
If OpenAI selection or summarization fails:
- Fall back to the top 1 to 3 rule-ranked candidates.
- Send a reduced output using available fields.
- Use original English title, source, and URL.
- Optionally include a brief fallback summary from `SummaryRaw` if it is clean enough.

The job should still try to deliver a digest unless no viable candidates remain.

### Telegram Failures
If Telegram delivery fails:
- Mark the run as failed.
- Log the Telegram API response.
- Optionally perform one conservative retry.
- Do not implement aggressive retry loops in the first version.

### Low-News Days
- Do not pad the digest.
- If only one or two items meet the threshold, send only those.
- If no items are strong enough, sending nothing is acceptable if logged clearly.

## Configuration
Configuration should be split between static source definitions and runtime secrets.

Static config examples:
- Enabled sources.
- Feed URLs or list-page URLs.
- Source type.
- Source-specific parser mode (`rss` or `html`).
- Scoring weights and keyword lists.

Runtime config examples:
- OpenAI API key.
- Telegram bot token.
- Telegram chat ID.

A simple config file such as YAML is sufficient for source metadata.

## Testing Strategy
The first version should target the highest-risk logic with focused tests.

Required tests:
- Title normalization and dedupe behavior.
- Rule-based scoring behavior.
- LLM structured-response parsing.
- Telegram message formatting.
- Source adapter fixture tests for selected RSS and HTML sources.

Testing objective:
- Protect fragile parsing and ranking logic.
- Keep tests deterministic.
- Avoid heavy integration test infrastructure in the first version.

## Proposed Project Structure
```text
ai-news-digest/
  cmd/
    ai-news-digest/
      main.go
  internal/
    config/
    model/
    source/
    pipeline/
    llm/
    deliver/
    format/
  configs/
    sources.yaml
  .github/
    workflows/
      daily-digest.yml
  docs/
    superpowers/
      specs/
        2026-03-21-ai-news-digest-design.md
```

## Open Questions for Implementation Planning
These do not block the design but should be finalized in the implementation plan:
- Exact feed or page URLs for each selected source.
- Which sources need HTML parsing rather than RSS.
- OpenAI model choice and prompt contract.
- Minimum score threshold for allowing an item into the digest.
- Telegram message size handling if summaries become too long.

## Recommended Implementation Sequence
1. Scaffold the Go CLI and configuration model.
2. Implement a small number of source adapters first: OpenAI, Anthropic, TechCrunch.
3. Build normalization, 24-hour filtering, dedupe, and scoring.
4. Add OpenAI candidate selection and summary generation.
5. Add Telegram delivery.
6. Add GitHub Actions workflow and secrets contract.
7. Expand source coverage and parser tests.

## Acceptance Criteria
The first version is successful when:
- GitHub Actions can run the tool manually and on schedule.
- The tool fetches from a curated source list and filters to the last 24 hours.
- Duplicate or overlapping stories are collapsed sensibly.
- The digest sends 1 to 3 items to Telegram.
- Each item contains English title, Chinese summary, why-it-matters explanation, source, and URL.
- Failures in a subset of sources do not break the full run.
- OpenAI failure still produces a reduced but usable digest when candidates exist.
