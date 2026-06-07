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

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second
			time.Sleep(backoff)
		}

		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				return fmt.Errorf("encode request: %w", err)
			}
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

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response: %w", readErr)
			continue
		}

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
			if resp.StatusCode == http.StatusNotFound {
				return ErrNotFound
			}
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

func (c *Client) AppendBlocksWithResponse(pageID string, req *AppendBlocksRequest) ([]Block, error) {
	var resp struct {
		Results []Block `json:"results"`
	}
	err := c.do("PATCH", "/blocks/"+pageID+"/children", req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c *Client) GetBlocks(pageID string) ([]Block, error) {
	blocks, err := c.fetchBlockPage(pageID, "")
	if err != nil {
		return nil, err
	}
	for i := range blocks {
		if blocks[i].HasChildren {
			children, err := c.GetBlocks(blocks[i].ID)
			if err != nil {
				return nil, fmt.Errorf("fetch children of %s: %w", blocks[i].ID, err)
			}
			blocks[i].Children = children
		}
	}
	return blocks, nil
}

func (c *Client) fetchBlockPage(pageID, cursor string) ([]Block, error) {
	path := "/blocks/" + pageID + "/children?page_size=100"
	if cursor != "" {
		path += "&start_cursor=" + cursor
	}
	var resp ListBlocksResponse
	if err := c.do("GET", path, nil, &resp); err != nil {
		return nil, err
	}
	if resp.HasMore {
		rest, err := c.fetchBlockPage(pageID, resp.NextCursor)
		if err != nil {
			return nil, err
		}
		return append(resp.Results, rest...), nil
	}
	return resp.Results, nil
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

func (c *Client) GetDatabase(databaseID string) (*Database, error) {
	var db Database
	err := c.do("GET", "/databases/"+databaseID, nil, &db)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (c *Client) QueryDatabase(databaseID string, req *QueryDatabaseRequest) (*QueryDatabaseResponse, error) {
	var resp QueryDatabaseResponse
	err := c.do("POST", "/databases/"+databaseID+"/query", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Status() error {
	req := SearchRequest{Query: "", PageSize: 1}
	var resp SearchResponse
	return c.do("POST", "/search", &req, &resp)
}
