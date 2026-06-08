# Notion Connector

The Notion connector allows VaultSync to synchronize notes with a Notion workspace.

## How It Works

Notes are converted from Markdown to Notion block format and pushed to Notion
via the Notion API. When pulling, Notion blocks are converted back to Markdown.

## Setup

1. Create an internal integration at https://www.notion.so/my-integrations
2. Copy the integration token
3. Share a target page with your integration
4. Run: `vault connect notion --token ntn_xxxxx --target-page-id <page-id>`

## Markdown → Notion Conversion

| Markdown | Notion Block |
|----------|-------------|
| `# Heading 1` | `heading_1` |
| `## Heading 2` | `heading_2` |
| `### Heading 3` | `heading_3` |
| Paragraph | `paragraph` |
| `- bullet` | `bulleted_list_item` |
| `1. item` | `numbered_list_item` |
| `- [ ] task` | `to_do` (unchecked) |
| `- [x] task` | `to_do` (checked) |
| `> quote` | `quote` |
| ` ```code``` ` | `code` |
| `---` | `divider` |
| Table | `table` + `table_row` |
| `**bold**` | `rich_text[bold]` |
| `*italic*` | `rich_text[italic]` |
| `[link](url)` | `rich_text[link]` |

## API Endpoints Used

- `POST /v1/pages` — Create page
- `PATCH /v1/pages/{id}` — Update properties
- `PATCH /v1/blocks/{id}` — Update content
- `GET /v1/blocks/{id}/children` — Fetch content
- `POST /v1/blocks/{id}/children` — Append blocks
- `POST /v1/search` — Find pages
- `GET /v1/databases/{id}/query` — Query database

## Rate Limiting

Notion enforces a rate limit of 3 requests per second. VaultSync handles this
with a sliding window limiter and exponential backoff on 429 responses.
