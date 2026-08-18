export interface CreateEventDto {
  title: string;
  calendar_id?: string;
  description?: string;
  start_at: string;
  end_at: string;
  timezone?: string;
  location?: string;
}

export interface UpdateEventDto {
  title?: string;
  calendar_id?: string;
  description?: string;
  start_at?: string;
  end_at?: string;
  timezone?: string;
  location?: string;
}

export interface EventQueryDto {
  calendar_id?: string;
  from?: string;
  to?: string;
}

export interface EventResponse {
  id: string;
  calendar_id: string;
  title: string;
  description: string | null;
  start_at: string;
  end_at: string;
  timezone: string;
  location: string | null;
  created_at: string;
  updated_at: string;
}
