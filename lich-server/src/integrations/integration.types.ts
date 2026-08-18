import type { Event } from '../db/repositories/event.repository.ts';

export interface ExternalCalendar {
  id: string;
  name: string;
  description?: string;
  timeZone?: string;
  isPrimary?: boolean;
  accessRole?: string;
}

export interface ExternalEventTime {
  dateTime?: string;
  date?: string;
  timeZone?: string;
}

export interface ExternalEvent {
  id: string;
  summary: string;
  description?: string;
  location?: string;
  start: ExternalEventTime;
  end: ExternalEventTime;
  status?: string; // 'confirmed' | 'tentative' | 'cancelled'
  updated?: string;
  etag?: string;
  htmlLink?: string;
}

export interface ExternalEventsResult {
  items: ExternalEvent[];
  nextSyncToken?: string;
  nextPageToken?: string;
}

export interface TokenResult {
  accessToken: string;
  refreshToken?: string | null;
  expiresIn?: number;
  tokenType: string;
  scope?: string;
  email?: string;
}

export interface CalendarProvider {
  readonly name: string;
  getAuthUrl(state: string): string;
  exchangeCode(code: string): Promise<TokenResult>;
  refreshToken(refreshToken: string): Promise<TokenResult>;
  listCalendars(accessToken: string): Promise<ExternalCalendar[]>;
  listEvents(accessToken: string, calendarId: string, syncToken?: string): Promise<ExternalEventsResult>;
  createEvent(accessToken: string, calendarId: string, event: Event): Promise<ExternalEvent>;
  updateEvent(accessToken: string, calendarId: string, externalId: string, event: Event): Promise<ExternalEvent>;
  deleteEvent(accessToken: string, calendarId: string, externalId: string): Promise<void>;
}
