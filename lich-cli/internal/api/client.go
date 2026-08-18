package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	cleanURL := strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: cleanURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request JSON: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	reqURL := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && (apiErr.ErrorName != "" || apiErr.Message != "") {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			ErrorName:  http.StatusText(resp.StatusCode),
			Message:    string(respBody),
		}
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w", err)
		}
	}

	return nil
}

func (c *Client) Health(ctx context.Context) error {
	var res map[string]string
	return c.doRequest(ctx, http.MethodGet, "/health", nil, &res)
}

func (c *Client) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	var res AuthResponse
	if err := c.doRequest(ctx, http.MethodPost, "/auth/register", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	var res AuthResponse
	if err := c.doRequest(ctx, http.MethodPost, "/auth/login", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) Me(ctx context.Context) (*User, error) {
	var res struct {
		User User `json:"user"`
	}
	if err := c.doRequest(ctx, http.MethodGet, "/auth/me", nil, &res); err != nil {
		return nil, err
	}
	return &res.User, nil
}

func (c *Client) ListCalendars(ctx context.Context) ([]Calendar, error) {
	var res []Calendar
	if err := c.doRequest(ctx, http.MethodGet, "/calendars", nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetCalendar(ctx context.Context, id string) (*Calendar, error) {
	var res Calendar
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/calendars/%s", id), nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) CreateCalendar(ctx context.Context, req CreateCalendarRequest) (*Calendar, error) {
	var res Calendar
	if err := c.doRequest(ctx, http.MethodPost, "/calendars", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) UpdateCalendar(ctx context.Context, id string, req UpdateCalendarRequest) (*Calendar, error) {
	var res Calendar
	if err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/calendars/%s", id), req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) DeleteCalendar(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/calendars/%s", id), nil, nil)
}

func (c *Client) ListEvents(ctx context.Context, filter EventFilter) ([]Event, error) {
	queryParams := url.Values{}
	if filter.CalendarID != "" {
		queryParams.Set("calendar_id", filter.CalendarID)
	}
	if filter.From != "" {
		queryParams.Set("from", filter.From)
	}
	if filter.To != "" {
		queryParams.Set("to", filter.To)
	}

	path := "/events"
	if len(queryParams) > 0 {
		path = fmt.Sprintf("%s?%s", path, queryParams.Encode())
	}

	var res []Event
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) GetEvent(ctx context.Context, id string) (*Event, error) {
	var res Event
	if err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/events/%s", id), nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	var res Event
	if err := c.doRequest(ctx, http.MethodPost, "/events", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) UpdateEvent(ctx context.Context, id string, req UpdateEventRequest) (*Event, error) {
	var res Event
	if err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/events/%s", id), req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) DeleteEvent(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/events/%s", id), nil, nil)
}
