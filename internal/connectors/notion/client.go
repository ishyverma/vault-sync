package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiBase        = "https://api.notion.com/v1"
	notionVersion  = "2022-06-28"
	rateLimit      = 3
	rateWindow     = time.Second
	maxRetries     = 3
)

type Client struct {
	token   string
	http    *http.Client
	ratelimit chan time.Time
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
		ratelimit: make(chan time.Time, rateLimit),
	}
}

func (c *Client) do(method, path string, body, out interface{}) error {
	c.waitRateLimit()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequest(method, apiBase+path, &buf)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Notion-Version", notionVersion)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := time.Second
			if v := resp.Header.Get("Retry-After"); v != "" {
				if d, err := time.ParseDuration(v + "s"); err == nil {
					retryAfter = d
				}
			}
			time.Sleep(retryAfter)
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %s", string(respBody))
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) waitRateLimit() {
	select {
	case c.ratelimit <- time.Now():
	default:
		oldest := <-c.ratelimit
		elapsed := time.Since(oldest)
		if elapsed < rateWindow {
			time.Sleep(rateWindow - elapsed)
		}
		c.ratelimit <- time.Now()
	}
}

func (c *Client) CreatePage(req *CreatePageRequest) (*Page, error) {
	var page Page
	err := c.do("POST", "/pages", req, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) UpdatePage(pageID string, req *UpdatePageRequest) (*Page, error) {
	var page Page
	err := c.do("PATCH", "/pages/"+pageID, req, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) GetPage(pageID string) (*Page, error) {
	var page Page
	err := c.do("GET", "/pages/"+pageID, nil, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) AppendBlocks(pageID string, req *AppendBlocksRequest) error {
	return c.do("PATCH", "/blocks/"+pageID+"/children", req, nil)
}

func (c *Client) GetBlocks(pageID string) ([]Block, error) {
	var blocks []Block
	cursor := ""
	for {
		path := "/blocks/" + pageID + "/children?page_size=100"
		if cursor != "" {
			path += "&start_cursor=" + cursor
		}
		var resp ListBlocksResponse
		if err := c.do("GET", path, nil, &resp); err != nil {
			return nil, err
		}
		blocks = append(blocks, resp.Results...)
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}
	return blocks, nil
}

func (c *Client) Search(query string) ([]Page, error) {
	req := SearchRequest{
		Query:    query,
		PageSize: 50,
	}
	var pages []Page
	cursor := ""
	for {
		req.StartCursor = cursor
		var resp SearchResponse
		if err := c.do("POST", "/search", &req, &resp); err != nil {
			return nil, err
		}
		pages = append(pages, resp.Results...)
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}
	return pages, nil
}

func (c *Client) DeletePage(pageID string) error {
	archived := true
	req := &UpdatePageRequest{Archived: &archived}
	return c.do("PATCH", "/pages/"+pageID, req, nil)
}

func (c *Client) DeleteBlock(blockID string) error {
	archived := true
	req := &UpdateBlockRequest{Archived: &archived}
	return c.do("PATCH", "/blocks/"+blockID, req, nil)
}

func (c *Client) Status() error {
	_, err := c.Search("")
	return err
}
