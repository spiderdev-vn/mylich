import type { Event } from '../../db/repositories/event.repository.ts';
import type {
  CalendarProvider,
  ExternalCalendar,
  ExternalEvent,
  ExternalEventsResult,
  TokenResult,
} from '../integration.types.ts';
import { GoogleEventMapper } from './google.mapper.ts';

export interface GoogleOAuthConfig {
  clientId: string;
  clientSecret: string;
  redirectUri: string;
}

export class GoogleProvider implements CalendarProvider {
  public readonly name = 'google';
  private config: GoogleOAuthConfig;

  constructor(config: GoogleOAuthConfig) {
    this.config = config;
  }

  public getAuthUrl(state: string): string {
    const scopes = [
      'https://www.googleapis.com/auth/calendar.events',
      'https://www.googleapis.com/auth/calendar.readonly',
      'https://www.googleapis.com/auth/userinfo.email',
    ].join(' ');

    const params = new URLSearchParams({
      client_id: this.config.clientId,
      redirect_uri: this.config.redirectUri,
      response_type: 'code',
      scope: scopes,
      access_type: 'offline',
      prompt: 'consent',
      state,
    });

    return `https://accounts.google.com/o/oauth2/v2/auth?${params.toString()}`;
  }

  public async exchangeCode(code: string): Promise<TokenResult> {
    const res = await fetch('https://oauth2.googleapis.com/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        code,
        client_id: this.config.clientId,
        client_secret: this.config.clientSecret,
        redirect_uri: this.config.redirectUri,
        grant_type: 'authorization_code',
      }),
    });

    if (!res.ok) {
      const errData = await res.text();
      throw new Error(`Google OAuth code exchange failed: ${res.status} ${errData}`);
    }

    const data = (await res.json()) as any;
    let email: string | undefined;

    // Fetch user info email if accessible
    try {
      const userRes = await fetch('https://www.googleapis.com/oauth2/v2/userinfo', {
        headers: { Authorization: `Bearer ${data.access_token}` },
      });
      if (userRes.ok) {
        const userData = (await userRes.json()) as any;
        email = userData.email;
      }
    } catch {
      // Non-fatal if userinfo fails
    }

    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token || null,
      expiresIn: data.expires_in,
      tokenType: data.token_type || 'Bearer',
      scope: data.scope,
      email,
    };
  }

  public async refreshToken(refreshToken: string): Promise<TokenResult> {
    const res = await fetch('https://oauth2.googleapis.com/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        refresh_token: refreshToken,
        client_id: this.config.clientId,
        client_secret: this.config.clientSecret,
        grant_type: 'refresh_token',
      }),
    });

    if (!res.ok) {
      const errData = await res.text();
      throw new Error(`Google OAuth token refresh failed: ${res.status} ${errData}`);
    }

    const data = (await res.json()) as any;
    return {
      accessToken: data.access_token,
      refreshToken: data.refresh_token || refreshToken,
      expiresIn: data.expires_in,
      tokenType: data.token_type || 'Bearer',
      scope: data.scope,
    };
  }

  public async listCalendars(accessToken: string): Promise<ExternalCalendar[]> {
    const res = await fetch('https://www.googleapis.com/calendar/v3/users/me/calendarList', {
      headers: { Authorization: `Bearer ${accessToken}` },
    });

    if (!res.ok) {
      throw new Error(`Failed to list Google calendars: ${res.status} ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return (data.items || []).map((item: any) => ({
      id: item.id,
      name: item.summary,
      description: item.description,
      timeZone: item.timeZone,
      isPrimary: Boolean(item.primary),
      accessRole: item.accessRole,
    }));
  }

  public async listEvents(
    accessToken: string,
    calendarId: string,
    syncToken?: string,
  ): Promise<ExternalEventsResult> {
    const params = new URLSearchParams();
    if (syncToken) {
      params.set('syncToken', syncToken);
    } else {
      params.set('singleEvents', 'true');
      params.set('maxResults', '250');
    }

    const encodedCalId = encodeURIComponent(calendarId);
    const res = await fetch(
      `https://www.googleapis.com/calendar/v3/calendars/${encodedCalId}/events?${params.toString()}`,
      {
        headers: { Authorization: `Bearer ${accessToken}` },
      },
    );

    if (!res.ok) {
      throw new Error(`Failed to fetch Google events: ${res.status} ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return {
      items: data.items || [],
      nextSyncToken: data.nextSyncToken,
      nextPageToken: data.nextPageToken,
    };
  }

  public async createEvent(
    accessToken: string,
    calendarId: string,
    event: Event,
  ): Promise<ExternalEvent> {
    const payload = GoogleEventMapper.toGoogle(event);
    const encodedCalId = encodeURIComponent(calendarId);

    const res = await fetch(
      `https://www.googleapis.com/calendar/v3/calendars/${encodedCalId}/events`,
      {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${accessToken}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      },
    );

    if (!res.ok) {
      throw new Error(`Failed to create Google event: ${res.status} ${await res.text()}`);
    }

    return (await res.json()) as ExternalEvent;
  }

  public async updateEvent(
    accessToken: string,
    calendarId: string,
    externalId: string,
    event: Event,
  ): Promise<ExternalEvent> {
    const payload = GoogleEventMapper.toGoogle(event);
    const encodedCalId = encodeURIComponent(calendarId);
    const encodedEventId = encodeURIComponent(externalId);

    const res = await fetch(
      `https://www.googleapis.com/calendar/v3/calendars/${encodedCalId}/events/${encodedEventId}`,
      {
        method: 'PATCH',
        headers: {
          Authorization: `Bearer ${accessToken}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      },
    );

    if (!res.ok) {
      throw new Error(`Failed to update Google event: ${res.status} ${await res.text()}`);
    }

    return (await res.json()) as ExternalEvent;
  }

  public async deleteEvent(
    accessToken: string,
    calendarId: string,
    externalId: string,
  ): Promise<void> {
    const encodedCalId = encodeURIComponent(calendarId);
    const encodedEventId = encodeURIComponent(externalId);

    const res = await fetch(
      `https://www.googleapis.com/calendar/v3/calendars/${encodedCalId}/events/${encodedEventId}`,
      {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${accessToken}` },
      },
    );

    if (!res.ok && res.status !== 404 && res.status !== 410) {
      throw new Error(`Failed to delete Google event: ${res.status} ${await res.text()}`);
    }
  }
}
