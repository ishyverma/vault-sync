---
title: "Comprehensive Go & Markdown Test"
date: 2026-06-06
tags:
  - go
  - testing
  - notion
  - markdown
  - complex
---

# H1: VaultSync Architecture Overview

This document serves as a **comprehensive** test for the *Markdown-to-Notion* converter. It includes *every* block type and inline formatting option.

## H2: Core Components

The system is built around three *core* abstractions: `Connector`, `Engine`, and `Store`. Each plays a **critical role** in the sync pipeline.

### H3: The Connector Interface

The `Connector` interface is the *heart* of the system — it defines how *different backends* (**Obsidian**, **Notion**, etc.) integrate:

```go
type Connector interface {
    Connect() error
    Push(note *storage.Note, content string, remoteID string) (string, error)
    Pull(remoteID string) (string, error)
    Delete(remoteID string) error
    Status() (bool, error)
    Name() string
}
```

> **Key insight:** Each backend implements this interface independently.

## Mixed Formatting Examples

Here's a paragraph with **bold**, *italic*, ***bold italic***, `inline code`, and a [link to GitHub](https://github.com/ishyverma/vault-sync). The converter should preserve *all* of these annotations when pushing to **Notion**.

Another paragraph with *nested `code` inside italic* and **`code inside bold`** — this tests annotation stacking.

## Lists

### Bulleted (Nested)

- Level 1 item with **bold text**
  - Level 2 with *italic text*
    - Level 3 with `inline code`
      - Level 4 — deep nesting
- Back to level 1 with a [link](https://example.com)
  - Level 2 again

### Numbered List

1. First step: **Initialize** the vault
   1. Sub-step with `config setup`
      1. Deep numbered item
2. Second step: *Connect* to Notion
   1. Get token from `notion.so/my-integrations`
3. Third step: Run `vault sync`

### Mixed Lists

1. Step one: Prepare
   - Bullet A under numbered
   - Bullet B under numbered
2. Step two: Execute
   - Detailed bullet
     1. Numbered under bullet
     2. Another numbered under bullet

### Todo Items

- [x] Implement `Connector` interface
- [x] Build **Notion client** with rate limiting
- [ ] Add *database mode* support
- [ ] Write `e2e tests` for the full pipeline
- [x] Fix the `rich_text` null bug

## Code Blocks

### Go (with syntax highlighting)

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// TokenBucket implements a simple rate limiter
type TokenBucket struct {
    tokens chan struct{}
    done   chan struct{}
}

func NewTokenBucket(rate int, interval time.Duration) *TokenBucket {
    tb := &TokenBucket{
        tokens: make(chan struct{}, rate),
        done:   make(chan struct{}),
    }

    go func() {
        ticker := time.NewTicker(interval / time.Duration(rate))
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                select {
                case tb.tokens <- struct{}{}:
                default:
                }
            case <-tb.done:
                return
            }
        }
    }()

    return tb
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
    select {
    case <-tb.tokens:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### JavaScript/JSON config example

```json
{
  "backends": {
    "notion": {
      "enabled": true,
      "token": "ntn_xxx...",
      "target_page_id": "37756705b6e5802dbbc1d6c45516a567"
    },
    "obsidian": {
      "enabled": true,
      "vault_path": "~/Documents/Obsidian/MyVault"
    }
  }
}
```

### Plain text / shell commands

```bash
# Initialize the vault
vault init

# Create a new note
vault new my-note.md --title "My Note" --template daily

# Push to all backends
vault sync --pull

# Check sync status
vault sync status
```

## Blockquotes & Callouts

> This is a **standard blockquote** with *formatting* and `inline code`.

> [!NOTE]
> This is a Notion-style callout with **bold** and *italic* text inside.

> [!WARNING]
> **Warning callout** — this should render as a callout in Notion.

## Tables

| Feature       | Status | Priority | Notes                  |
|---------------|--------|----------|------------------------|
| Markdown→Notion | ✅ Done | High     | *Inline formatting* supported |
| Notion→Markdown | ✅ Done | High     | `rich_text` annotations |
| Database mode    | ❌ TODO | Medium   | Blocked by API design |
| E2E tests        | ❌ TODO | High     | Needs **CI setup**    |

## Thematic Break

---

## Complex Paragraph with Multiple Inline Elements

Consider a scenario where we have `extractRichText` walking the **Goldmark AST** — it needs to handle *nested emphasis* (like ***this***), `inline code with **bold** inside` (actually not possible in standard Markdown, but good to test), and a [reference link](https://opencode.ai). The converter should also preserve **trailing bold text**.

```go
// Edge case: multiple annotations on the same rich text
rt := []RichText{
    {Type: "text", Text: &TextContent{Content: "bold "}, Annotations: &Annotations{Bold: true}},
    {Type: "text", Text: &TextContent{Content: "and "}},
    {Type: "text", Text: &TextContent{Content: "italic"}, Annotations: &Annotations{Italic: true}},
}
```

## Edge Cases

### Empty Content
A paragraph with only a single word.

### List with One Item

- Single bullet

### Nested Blockquote

> Outer quote
> > Inner quote with **formatting**
> > > Deepest level

### HTML (should be escaped or stripped)

<div>This should not break the converter</div>

## Conclusion

This test file exercises **every** block type supported by the `MarkdownToBlocks` converter:

1. ✅ **Headings** (H1, H2, H3)
2. ✅ **Paragraphs** with inline formatting
3. ✅ **Bulleted lists** (nested)
4. ✅ **Numbered lists** (nested)
5. ✅ **Todo items** (checked/unchecked)
6. ✅ **Code blocks** with language
7. ✅ **Blockquotes** (nested)
8. ✅ **Callouts** (`[!NOTE]`)
9. ✅ **Tables** with headers
10. ✅ **Thematic breaks**
11. ✅ **Inline formatting** (bold, italic, code, links)

> [!IMPORTANT]
> After pushing this note to Notion via `vault push test-complex.md`, verify **all** the above elements render correctly. If anything looks off, the converter needs a fix.
