export interface CreateCalendarDto {
  name: string;
  description?: string;
  timezone?: string;
}

export interface UpdateCalendarDto {
  name?: string;
  description?: string;
  timezone?: string;
  is_default?: boolean;
}

export interface CalendarResponse {
  id: string;
  user_id: string;
  name: string;
  description: string | null;
  timezone: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}
