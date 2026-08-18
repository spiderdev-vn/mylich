import { BadRequestError } from './errors.ts';

export function isValidTimezone(timezone: string): boolean {
  if (!timezone || typeof timezone !== 'string') {
    return false;
  }
  try {
    Intl.DateTimeFormat(undefined, { timeZone: timezone });
    return true;
  } catch {
    return false;
  }
}

export function validateTimezone(timezone: string): void {
  if (!isValidTimezone(timezone)) {
    throw new BadRequestError(`Invalid IANA timezone: '${timezone}'`);
  }
}

export function isValidIsoDate(dateString: string): boolean {
  if (!dateString || typeof dateString !== 'string') {
    return false;
  }
  const date = new Date(dateString);
  return !isNaN(date.getTime()) && dateString.includes('T');
}

export function validateEventDates(startAt: string, endAt: string): { start: Date; end: Date } {
  if (!isValidIsoDate(startAt)) {
    throw new BadRequestError(`Invalid start_at date-time format: '${startAt}'. Expected ISO 8601 string.`);
  }
  if (!isValidIsoDate(endAt)) {
    throw new BadRequestError(`Invalid end_at date-time format: '${endAt}'. Expected ISO 8601 string.`);
  }

  const start = new Date(startAt);
  const end = new Date(endAt);

  if (end.getTime() <= start.getTime()) {
    throw new BadRequestError(`end_at (${endAt}) must be after start_at (${startAt})`);
  }

  return { start, end };
}
