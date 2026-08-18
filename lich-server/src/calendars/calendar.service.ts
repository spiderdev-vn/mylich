import crypto from 'node:crypto';
import type { CalendarRepository, Calendar } from '../db/repositories/calendar.repository.ts';
import type { CreateCalendarDto, UpdateCalendarDto, CalendarResponse } from './calendar.types.ts';
import { BadRequestError, NotFoundError } from '../common/errors.ts';
import { validateTimezone } from '../common/timezone.ts';

export class CalendarService {
  private calendarRepo: CalendarRepository;

  constructor(calendarRepo: CalendarRepository) {
    this.calendarRepo = calendarRepo;
  }

  private mapToResponse(cal: Calendar): CalendarResponse {
    return {
      id: cal.id,
      user_id: cal.user_id,
      name: cal.name,
      description: cal.description,
      timezone: cal.timezone,
      is_default: cal.is_default === 1,
      created_at: cal.created_at,
      updated_at: cal.updated_at,
    };
  }

  public create(userId: string, input: CreateCalendarDto): CalendarResponse {
    if (!input.name || typeof input.name !== 'string' || input.name.trim().length === 0) {
      throw new BadRequestError('Calendar name is required');
    }

    const timezone = input.timezone || 'UTC';
    validateTimezone(timezone);

    const id = crypto.randomUUID();
    const calendar = this.calendarRepo.create({
      id,
      user_id: userId,
      name: input.name.trim(),
      description: input.description?.trim() || null,
      timezone,
      is_default: false,
    });

    return this.mapToResponse(calendar);
  }

  public listByUser(userId: string): CalendarResponse[] {
    const calendars = this.calendarRepo.findByUserId(userId);
    return calendars.map((c) => this.mapToResponse(c));
  }

  public getById(id: string, userId: string): CalendarResponse {
    const calendar = this.calendarRepo.findById(id);
    if (!calendar) {
      throw new NotFoundError(`Calendar with id '${id}' not found`);
    }
    if (calendar.user_id !== userId) {
      throw new NotFoundError(`Calendar with id '${id}' not found`);
    }
    return this.mapToResponse(calendar);
  }

  public update(id: string, userId: string, input: UpdateCalendarDto): CalendarResponse {
    const calendar = this.calendarRepo.findById(id);
    if (!calendar || calendar.user_id !== userId) {
      throw new NotFoundError(`Calendar with id '${id}' not found`);
    }

    if (input.name !== undefined && input.name.trim().length === 0) {
      throw new BadRequestError('Calendar name cannot be empty');
    }

    if (input.timezone !== undefined) {
      validateTimezone(input.timezone);
    }

    const updated = this.calendarRepo.update(id, {
      name: input.name?.trim(),
      description: input.description !== undefined ? input.description.trim() : undefined,
      timezone: input.timezone,
      is_default: input.is_default !== undefined ? (input.is_default ? 1 : 0) : undefined,
    });

    if (!updated) {
      throw new NotFoundError(`Calendar with id '${id}' not found`);
    }

    return this.mapToResponse(updated);
  }

  public delete(id: string, userId: string): void {
    const calendar = this.calendarRepo.findById(id);
    if (!calendar || calendar.user_id !== userId) {
      throw new NotFoundError(`Calendar with id '${id}' not found`);
    }

    this.calendarRepo.delete(id);
  }
}
