import crypto from 'node:crypto';
import type { EventRepository, Event } from '../db/repositories/event.repository.ts';
import type { CalendarRepository } from '../db/repositories/calendar.repository.ts';
import type {
  CreateEventDto,
  UpdateEventDto,
  EventQueryDto,
  EventResponse,
} from './event.types.ts';
import { BadRequestError, NotFoundError } from '../common/errors.ts';
import { validateTimezone, validateEventDates, isValidIsoDate } from '../common/timezone.ts';

export class EventService {
  private eventRepo: EventRepository;
  private calendarRepo: CalendarRepository;

  constructor(eventRepo: EventRepository, calendarRepo: CalendarRepository) {
    this.eventRepo = eventRepo;
    this.calendarRepo = calendarRepo;
  }

  private mapToResponse(event: Event): EventResponse {
    return {
      id: event.id,
      calendar_id: event.calendar_id,
      title: event.title,
      description: event.description,
      start_at: event.start_at,
      end_at: event.end_at,
      timezone: event.timezone,
      location: event.location,
      created_at: event.created_at,
      updated_at: event.updated_at,
    };
  }

  public create(userId: string, input: CreateEventDto): EventResponse {
    if (!input.title || typeof input.title !== 'string' || input.title.trim().length === 0) {
      throw new BadRequestError('Event title is required');
    }

    if (!input.start_at || !input.end_at) {
      throw new BadRequestError('start_at and end_at are required');
    }

    validateEventDates(input.start_at, input.end_at);

    // Resolve calendar
    let calendarId = input.calendar_id;
    let calendarTimezone = 'UTC';

    if (calendarId) {
      const cal = this.calendarRepo.findById(calendarId);
      if (!cal || cal.user_id !== userId) {
        throw new BadRequestError(`Calendar with id '${calendarId}' not found`);
      }
      calendarTimezone = cal.timezone;
    } else {
      const defaultCal = this.calendarRepo.findDefaultByUserId(userId);
      if (defaultCal) {
        calendarId = defaultCal.id;
        calendarTimezone = defaultCal.timezone;
      } else {
        const userCals = this.calendarRepo.findByUserId(userId);
        if (userCals.length > 0) {
          calendarId = userCals[0].id;
          calendarTimezone = userCals[0].timezone;
        } else {
          throw new BadRequestError('No calendar available for user');
        }
      }
    }

    const timezone = input.timezone || calendarTimezone;
    validateTimezone(timezone);

    const id = crypto.randomUUID();
    const event = this.eventRepo.create({
      id,
      calendar_id: calendarId,
      title: input.title.trim(),
      description: input.description?.trim() || null,
      start_at: input.start_at,
      end_at: input.end_at,
      timezone,
      location: input.location?.trim() || null,
    });

    return this.mapToResponse(event);
  }

  public list(userId: string, query: EventQueryDto): EventResponse[] {
    if (query.from && !isValidIsoDate(query.from)) {
      throw new BadRequestError(`Invalid from date: '${query.from}'. Expected ISO 8601 string.`);
    }
    if (query.to && !isValidIsoDate(query.to)) {
      throw new BadRequestError(`Invalid to date: '${query.to}'. Expected ISO 8601 string.`);
    }

    if (query.calendar_id) {
      const cal = this.calendarRepo.findById(query.calendar_id);
      if (!cal || cal.user_id !== userId) {
        throw new NotFoundError(`Calendar with id '${query.calendar_id}' not found`);
      }
    }

    const events = this.eventRepo.findEvents({
      userId,
      calendarId: query.calendar_id,
      from: query.from,
      to: query.to,
    });

    return events.map((e) => this.mapToResponse(e));
  }

  public getById(id: string, userId: string): EventResponse {
    const eventWithCal = this.eventRepo.findByIdWithCalendar(id);
    if (!eventWithCal || eventWithCal.user_id !== userId) {
      throw new NotFoundError(`Event with id '${id}' not found`);
    }

    return this.mapToResponse(eventWithCal);
  }

  public update(id: string, userId: string, input: UpdateEventDto): EventResponse {
    const existing = this.eventRepo.findByIdWithCalendar(id);
    if (!existing || existing.user_id !== userId) {
      throw new NotFoundError(`Event with id '${id}' not found`);
    }

    if (input.title !== undefined && input.title.trim().length === 0) {
      throw new BadRequestError('Event title cannot be empty');
    }

    let newCalendarId = existing.calendar_id;
    if (input.calendar_id !== undefined && input.calendar_id !== existing.calendar_id) {
      const newCal = this.calendarRepo.findById(input.calendar_id);
      if (!newCal || newCal.user_id !== userId) {
        throw new BadRequestError(`Target calendar '${input.calendar_id}' not found`);
      }
      newCalendarId = input.calendar_id;
    }

    const startAt = input.start_at || existing.start_at;
    const endAt = input.end_at || existing.end_at;
    validateEventDates(startAt, endAt);

    if (input.timezone !== undefined) {
      validateTimezone(input.timezone);
    }

    const updated = this.eventRepo.update(id, {
      title: input.title?.trim(),
      calendar_id: newCalendarId,
      description: input.description !== undefined ? input.description.trim() : undefined,
      start_at: input.start_at,
      end_at: input.end_at,
      timezone: input.timezone,
      location: input.location !== undefined ? input.location.trim() : undefined,
    });

    if (!updated) {
      throw new NotFoundError(`Event with id '${id}' not found`);
    }

    return this.mapToResponse(updated);
  }

  public delete(id: string, userId: string): void {
    const existing = this.eventRepo.findByIdWithCalendar(id);
    if (!existing || existing.user_id !== userId) {
      throw new NotFoundError(`Event with id '${id}' not found`);
    }

    this.eventRepo.softDelete(id);
  }
}
