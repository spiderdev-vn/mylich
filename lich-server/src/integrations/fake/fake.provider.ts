import type { Event } from '../../db/repositories/event.repository.ts';
import type {
  CalendarProvider,
  ExternalCalendar,
  ExternalEvent,
  ExternalEventsResult,
  TokenResult,
} from '../integration.types.ts';
import { GoogleEventMapper } from '../google/google.mapper.ts';

export class FakeGoogleProvider implements CalendarProvider {
  public readonly name = 'google';
  public calendars: ExternalCalendar[] = [
    {
      id: 'primary',
      name: 'Personal (Fake Google)',
      description: 'Main personal calendar',
      timeZone: 'Asia/Ho_Chi_Minh',
      isPrimary: true,
      accessRole: 'owner',
    },
    {
      id: 'work@example.com',
      name: 'Work (Fake Google)',
      description: 'Work meetings',
      timeZone: 'Asia/Ho_Chi_Minh',
      isPrimary: false,
      accessRole: 'owner',
    },
  ];

  public eventsByCalendar: Map<string, ExternalEvent[]> = new Map();
  public syncCounter = 1;

  constructor() {
    this.eventsByCalendar.set('primary', []);
    this.eventsByCalendar.set('work@example.com', []);
  }

  public getAuthUrl(state: string): string {
    return `http://127.0.0.1:3000/auth/google/callback?code=fake-auth-code-123&state=${encodeURIComponent(state)}`;
  }

  public async exchangeCode(code: string): Promise<TokenResult> {
    return {
      accessToken: `fake-access-token-${code}`,
      refreshToken: 'fake-refresh-token-xyz',
      expiresIn: 3600,
      tokenType: 'Bearer',
      scope: 'https://www.googleapis.com/auth/calendar.events',
      email: 'user@gmail.com',
    };
  }

  public async refreshToken(refreshToken: string): Promise<TokenResult> {
    return {
      accessToken: `fake-refreshed-token-${Date.now()}`,
      refreshToken,
      expiresIn: 3600,
      tokenType: 'Bearer',
    };
  }

  public async listCalendars(_accessToken: string): Promise<ExternalCalendar[]> {
    return this.calendars;
  }

  public async createCalendar(
    _accessToken: string,
    name: string,
    timeZone = 'UTC',
  ): Promise<ExternalCalendar> {
    const id = `cal-${Date.now()}@group.calendar.google.com`;
    const cal: ExternalCalendar = {
      id,
      name,
      description: 'Created from Lich',
      timeZone,
      isPrimary: false,
      accessRole: 'owner',
    };
    this.calendars.push(cal);
    this.eventsByCalendar.set(id, []);
    return cal;
  }

  public async listEvents(
    _accessToken: string,
    calendarId: string,
    _syncToken?: string,
    _timeMin?: string,
    _timeMax?: string,
  ): Promise<ExternalEventsResult> {
    const list = this.eventsByCalendar.get(calendarId) || [];
    this.syncCounter++;
    return {
      items: [...list],
      nextSyncToken: `sync-token-${this.syncCounter}`,
    };
  }

  public async createEvent(
    _accessToken: string,
    calendarId: string,
    event: Event,
  ): Promise<ExternalEvent> {
    const mapped = GoogleEventMapper.toGoogle(event);
    const extEvent: ExternalEvent = {
      id: `g-evt-${Date.now()}-${Math.random().toString(36).substring(7)}`,
      summary: mapped.summary || event.title,
      description: mapped.description,
      location: mapped.location,
      start: mapped.start || { dateTime: event.start_at },
      end: mapped.end || { dateTime: event.end_at },
      status: 'confirmed',
      updated: new Date().toISOString(),
    };

    const list = this.eventsByCalendar.get(calendarId) || [];
    list.push(extEvent);
    this.eventsByCalendar.set(calendarId, list);

    return extEvent;
  }

  public async updateEvent(
    _accessToken: string,
    calendarId: string,
    externalId: string,
    event: Event,
  ): Promise<ExternalEvent> {
    const list = this.eventsByCalendar.get(calendarId) || [];
    const idx = list.findIndex((e) => e.id === externalId);
    const mapped = GoogleEventMapper.toGoogle(event);

    const updated: ExternalEvent = {
      id: externalId,
      summary: mapped.summary || event.title,
      description: mapped.description,
      location: mapped.location,
      start: mapped.start || { dateTime: event.start_at },
      end: mapped.end || { dateTime: event.end_at },
      status: 'confirmed',
      updated: new Date().toISOString(),
    };

    if (idx >= 0) {
      list[idx] = updated;
    } else {
      list.push(updated);
    }
    this.eventsByCalendar.set(calendarId, list);

    return updated;
  }

  public async deleteEvent(
    _accessToken: string,
    calendarId: string,
    externalId: string,
  ): Promise<void> {
    const list = this.eventsByCalendar.get(calendarId) || [];
    const filtered = list.filter((e) => e.id !== externalId);
    this.eventsByCalendar.set(calendarId, filtered);
  }
}
