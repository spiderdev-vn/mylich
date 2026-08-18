package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Health(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Login(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/login" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var req LoginRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Username != "testuser" || req.Password != "pass123" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid username or password"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"mock-jwt-token","user":{"id":"user-1","username":"testuser"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	res, err := client.Login(context.Background(), LoginRequest{
		Username: "testuser",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Token != "mock-jwt-token" {
		t.Errorf("expected token 'mock-jwt-token', got %s", res.Token)
	}

	// Test invalid login
	_, err = client.Login(context.Background(), LoginRequest{
		Username: "testuser",
		Password: "wrong",
	})
	if err == nil {
		t.Fatalf("expected error for invalid credentials, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.StatusCode != 401 {
		t.Errorf("expected APIError with status 401, got %v", err)
	}
}

func TestClient_EventsCRUD(t *testing.T) {
	eventsDB := make(map[string]Event)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/events":
			var req CreateEventRequest
			json.NewDecoder(r.Body).Decode(&req)
			event := Event{
				ID:         "event-123",
				Title:      req.Title,
				StartAt:    req.StartAt,
				EndAt:      req.EndAt,
				CalendarID: "cal-1",
			}
			eventsDB[event.ID] = event
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(event)

		case r.Method == http.MethodGet && r.URL.Path == "/events":
			list := make([]Event, 0, len(eventsDB))
			for _, e := range eventsDB {
				list = append(list, e)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(list)

		case r.Method == http.MethodDelete && r.URL.Path == "/events/event-123":
			delete(eventsDB, "event-123")
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "valid-token")

	// 1. Create Event
	created, err := client.CreateEvent(context.Background(), CreateEventRequest{
		Title:   "Sprint Planning",
		StartAt: "2026-08-18T10:00:00Z",
		EndAt:   "2026-08-18T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}
	if created.ID != "event-123" || created.Title != "Sprint Planning" {
		t.Errorf("unexpected created event: %+v", created)
	}

	// 2. List Events
	list, err := client.ListEvents(context.Background(), EventFilter{})
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 event, got %d", len(list))
	}

	// 3. Delete Event
	err = client.DeleteEvent(context.Background(), "event-123")
	if err != nil {
		t.Fatalf("failed to delete event: %v", err)
	}

	// 4. Verify empty
	list, _ = client.ListEvents(context.Background(), EventFilter{})
	if len(list) != 0 {
		t.Errorf("expected 0 events after delete, got %d", len(list))
	}
}
