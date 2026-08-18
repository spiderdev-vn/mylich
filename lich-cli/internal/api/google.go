package api

import (
	"context"
	"net/http"
)

type GoogleAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
}

type GoogleMappedCalendar struct {
	CalendarID         string `json:"calendarId"`
	CalendarName       string `json:"calendarName"`
	ExternalCalendarID string `json:"externalCalendarId"`
	SyncDirection      string `json:"syncDirection"`
	LastSyncedAt       string `json:"lastSyncedAt,omitempty"`
}

type GoogleStatusResponse struct {
	Connected                bool                   `json:"connected"`
	Provider                 string                 `json:"provider"`
	Status                   string                 `json:"status"`
	Email                    string                 `json:"email,omitempty"`
	MappedCalendars          []GoogleMappedCalendar `json:"mappedCalendars"`
	UnresolvedConflictsCount int                    `json:"unresolvedConflictsCount"`
}

type GoogleExternalCalendar struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
	IsPrimary   bool   `json:"isPrimary"`
	AccessRole  string `json:"accessRole,omitempty"`
}

type GoogleCalendarsResponse struct {
	Calendars []GoogleExternalCalendar `json:"calendars"`
}

type GoogleMapRequest struct {
	CalendarID         string `json:"calendar_id"`
	ExternalCalendarID string `json:"external_calendar_id"`
	SyncDirection      string `json:"sync_direction,omitempty"`
}

type GoogleSyncRequest struct {
	CalendarID string `json:"calendar_id,omitempty"`
	Direction  string `json:"direction,omitempty"`
}

type GoogleSyncResponse struct {
	Success bool `json:"success"`
	Pushed  int  `json:"pushed"`
	Pulled  int  `json:"pulled"`
}

func (c *Client) GetGoogleAuthURL(ctx context.Context) (string, error) {
	var res GoogleAuthURLResponse
	if err := c.doRequest(ctx, http.MethodGet, "/integrations/google/auth-url", nil, &res); err != nil {
		return "", err
	}
	return res.AuthURL, nil
}

func (c *Client) GetGoogleStatus(ctx context.Context) (*GoogleStatusResponse, error) {
	var res GoogleStatusResponse
	if err := c.doRequest(ctx, http.MethodGet, "/integrations/google/status", nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) ListGoogleCalendars(ctx context.Context) ([]GoogleExternalCalendar, error) {
	var res GoogleCalendarsResponse
	if err := c.doRequest(ctx, http.MethodGet, "/integrations/google/calendars", nil, &res); err != nil {
		return nil, err
	}
	return res.Calendars, nil
}

func (c *Client) MapGoogleCalendar(ctx context.Context, calendarID, externalCalendarID, syncDirection string) error {
	req := GoogleMapRequest{
		CalendarID:         calendarID,
		ExternalCalendarID: externalCalendarID,
		SyncDirection:      syncDirection,
	}
	var res map[string]any
	return c.doRequest(ctx, http.MethodPost, "/integrations/google/map", req, &res)
}

func (c *Client) SyncGoogle(ctx context.Context, calendarID, direction string) (*GoogleSyncResponse, error) {
	req := GoogleSyncRequest{
		CalendarID: calendarID,
		Direction:  direction,
	}
	var res GoogleSyncResponse
	if err := c.doRequest(ctx, http.MethodPost, "/integrations/google/sync", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) DisconnectGoogle(ctx context.Context) error {
	var res map[string]any
	return c.doRequest(ctx, http.MethodDelete, "/integrations/google", nil, &res)
}
