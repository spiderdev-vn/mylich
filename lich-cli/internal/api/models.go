package api

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Timezone string `json:"timezone,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Calendar struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Timezone    string `json:"timezone"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateCalendarRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

type UpdateCalendarRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	IsDefault   *bool  `json:"is_default,omitempty"`
}

type Event struct {
	ID          string `json:"id"`
	CalendarID  string `json:"calendar_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
	Timezone    string `json:"timezone"`
	Location    string `json:"location"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateEventRequest struct {
	Title       string `json:"title"`
	CalendarID  string `json:"calendar_id,omitempty"`
	Description string `json:"description,omitempty"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
	Timezone    string `json:"timezone,omitempty"`
	Location    string `json:"location,omitempty"`
}

type UpdateEventRequest struct {
	Title       string `json:"title,omitempty"`
	CalendarID  string `json:"calendar_id,omitempty"`
	Description string `json:"description,omitempty"`
	StartAt     string `json:"start_at,omitempty"`
	EndAt       string `json:"end_at,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	Location    string `json:"location,omitempty"`
}

type EventFilter struct {
	CalendarID string
	From       string
	To         string
}

type APIError struct {
	StatusCode int    `json:"statusCode"`
	ErrorName  string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.ErrorName
}
