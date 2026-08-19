import type { Event } from '../../db/repositories/event.repository.ts';
import type { ExternalEvent } from '../integration.types.ts';

export class GoogleEventMapper {
  /**
   * Convert a Lich Event to a Google Calendar Event payload
   */
  public static toGoogle(event: Event): Partial<ExternalEvent> {
    const tz = event.timezone || 'UTC';

    let startIso = event.start_at;
    let endIso = event.end_at;
    try {
      startIso = new Date(event.start_at).toISOString();
      endIso = new Date(event.end_at).toISOString();
    } catch {
      // fallback
    }

    return {
      summary: event.title,
      description: event.description || '',
      location: event.location || '',
      start: {
        dateTime: startIso,
        timeZone: tz,
      },
      end: {
        dateTime: endIso,
        timeZone: tz,
      },
    };
  }

  /**
   * Convert a Google Calendar Event to Lich Event fields
   */
  public static toLich(
    googleEvent: ExternalEvent,
    _calendarId?: string,
  ): {
    title: string;
    description: string;
    start_at: string;
    end_at: string;
    timezone: string;
    location: string;
  } {
    const title = googleEvent.summary || '(Không có tiêu đề)';
    const description = googleEvent.description || '';
    const location = googleEvent.location || '';

    let startAt = '';
    let endAt = '';
    let tz = 'UTC';

    if (googleEvent.start.dateTime && googleEvent.end.dateTime) {
      // Timed event
      startAt = new Date(googleEvent.start.dateTime).toISOString();
      endAt = new Date(googleEvent.end.dateTime).toISOString();
      tz = googleEvent.start.timeZone || 'UTC';
    } else if (googleEvent.start.date && googleEvent.end.date) {
      // All-day event (Google returns 'YYYY-MM-DD')
      startAt = `${googleEvent.start.date}T00:00:00.000Z`;
      endAt = `${googleEvent.end.date}T00:00:00.000Z`;
      tz = 'UTC';
    } else {
      // Fallback
      startAt = new Date().toISOString();
      endAt = new Date(Date.now() + 3600000).toISOString();
    }

    return {
      title,
      description,
      start_at: startAt,
      end_at: endAt,
      timezone: tz,
      location,
    };
  }
}
