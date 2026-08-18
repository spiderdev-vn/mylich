import crypto from 'node:crypto';
import type { IntegrationRepository } from '../db/repositories/integration.repository.ts';
import type { CalendarIntegrationRepository } from '../db/repositories/calendar_integration.repository.ts';
import type { EventIntegrationRepository } from '../db/repositories/event_integration.repository.ts';
import type { EventRepository } from '../db/repositories/event.repository.ts';
import type { CalendarRepository } from '../db/repositories/calendar.repository.ts';
import type { CalendarProvider, ExternalCalendar } from './integration.types.ts';
import { GoogleEventMapper } from './google/google.mapper.ts';
import { BadRequestError, NotFoundError } from '../common/errors.ts';

export interface IntegrationStatusResponse {
  connected: boolean;
  provider: string;
  status: string;
  email?: string;
  mappedCalendars: {
    calendarId: string;
    calendarName: string;
    externalCalendarId: string;
    syncDirection: string;
    lastSyncedAt?: string;
  }[];
  unresolvedConflictsCount: number;
}

export class IntegrationService {
  private integrationRepo: IntegrationRepository;
  private calendarIntegrationRepo: CalendarIntegrationRepository;
  private eventIntegrationRepo: EventIntegrationRepository;
  private eventRepo: EventRepository;
  private calendarRepo: CalendarRepository;
  private provider: CalendarProvider;

  constructor(
    integrationRepo: IntegrationRepository,
    calendarIntegrationRepo: CalendarIntegrationRepository,
    eventIntegrationRepo: EventIntegrationRepository,
    eventRepo: EventRepository,
    calendarRepo: CalendarRepository,
    provider: CalendarProvider,
  ) {
    this.integrationRepo = integrationRepo;
    this.calendarIntegrationRepo = calendarIntegrationRepo;
    this.eventIntegrationRepo = eventIntegrationRepo;
    this.eventRepo = eventRepo;
    this.calendarRepo = calendarRepo;
    this.provider = provider;
  }

  public getAuthUrl(userId: string): { authUrl: string } {
    const statePayload = Buffer.from(JSON.stringify({ userId, nonce: crypto.randomUUID() })).toString('base64url');
    const authUrl = this.provider.getAuthUrl(statePayload);
    return { authUrl };
  }

  public async handleCallback(userId: string, code: string): Promise<{ success: boolean; email?: string }> {
    const tokenResult = await this.provider.exchangeCode(code);

    const integrationId = crypto.randomUUID();
    const integration = this.integrationRepo.upsertIntegration({
      id: integrationId,
      user_id: userId,
      provider: this.provider.name,
      status: 'connected',
    });

    this.integrationRepo.saveCredentials({
      integration_id: integration.id,
      access_token: tokenResult.accessToken,
      refresh_token: tokenResult.refreshToken,
      token_type: tokenResult.tokenType,
      expires_at: tokenResult.expiresIn
        ? new Date(Date.now() + tokenResult.expiresIn * 1000).toISOString()
        : null,
      scope: tokenResult.scope,
    });

    // Auto-map default primary calendar if available
    const defaultCal = this.calendarRepo.findDefaultByUserId(userId);
    if (defaultCal) {
      const existingMapping = this.calendarIntegrationRepo.findByCalendarAndIntegration(
        defaultCal.id,
        integration.id,
      );
      if (!existingMapping) {
        this.calendarIntegrationRepo.upsert({
          id: crypto.randomUUID(),
          calendar_id: defaultCal.id,
          integration_id: integration.id,
          external_calendar_id: 'primary',
          sync_direction: 'bidirectional',
          enabled: true,
        });
      }
    }

    return { success: true, email: tokenResult.email };
  }

  public async getStatus(userId: string): Promise<IntegrationStatusResponse> {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (!integration || integration.status === 'disconnected') {
      return {
        connected: false,
        provider: this.provider.name,
        status: 'disconnected',
        mappedCalendars: [],
        unresolvedConflictsCount: 0,
      };
    }

    const mappings = this.calendarIntegrationRepo.listByIntegrationId(integration.id);
    const conflicts = this.eventIntegrationRepo.listUnresolvedConflicts(userId);

    const mappedCalendars = mappings.map((m) => {
      const cal = this.calendarRepo.findById(m.calendar_id);
      const syncState = this.calendarIntegrationRepo.getSyncState(
        integration.id,
        `calendar:${m.external_calendar_id}`,
      );

      return {
        calendarId: m.calendar_id,
        calendarName: cal?.name || m.calendar_id,
        externalCalendarId: m.external_calendar_id,
        syncDirection: m.sync_direction,
        lastSyncedAt: syncState?.last_synced_at,
      };
    });

    return {
      connected: integration.status === 'connected',
      provider: this.provider.name,
      status: integration.status,
      mappedCalendars,
      unresolvedConflictsCount: conflicts.length,
    };
  }

  public async listExternalCalendars(userId: string): Promise<ExternalCalendar[]> {
    const accessToken = await this.getValidAccessToken(userId);
    return this.provider.listCalendars(accessToken);
  }

  public mapCalendar(
    userId: string,
    calendarId: string,
    externalCalendarId: string,
    syncDirection?: 'push' | 'pull' | 'bidirectional',
  ): void {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (!integration || integration.status !== 'connected') {
      throw new BadRequestError('Integration is not connected');
    }

    const cal = this.calendarRepo.findById(calendarId);
    if (!cal || cal.user_id !== userId) {
      throw new NotFoundError(`Calendar '${calendarId}' not found`);
    }

    this.calendarIntegrationRepo.upsert({
      id: crypto.randomUUID(),
      calendar_id: calendarId,
      integration_id: integration.id,
      external_calendar_id: externalCalendarId,
      sync_direction: syncDirection || 'bidirectional',
      enabled: true,
    });
  }

  public async createAndMapCalendar(
    userId: string,
    calendarId: string,
    name?: string,
    syncDirection?: 'push' | 'pull' | 'bidirectional',
  ): Promise<ExternalCalendar> {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (!integration || integration.status !== 'connected') {
      throw new BadRequestError('Google integration is not connected. Connect first with "lich google connect"');
    }

    const cal = this.calendarRepo.findById(calendarId);
    if (!cal || cal.user_id !== userId) {
      throw new NotFoundError(`Calendar '${calendarId}' not found`);
    }

    const accessToken = await this.getValidAccessToken(userId);
    const targetName = name || cal.name;
    const extCal = await this.provider.createCalendar(accessToken, targetName, cal.timezone);

    this.calendarIntegrationRepo.upsert({
      id: crypto.randomUUID(),
      calendar_id: calendarId,
      integration_id: integration.id,
      external_calendar_id: extCal.id,
      sync_direction: syncDirection || 'bidirectional',
      enabled: true,
    });

    return extCal;
  }

  public async sync(
    userId: string,
    calendarId?: string,
    direction: 'push' | 'pull' | 'both' = 'both',
  ): Promise<{ pushed: number; pulled: number }> {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (!integration || integration.status !== 'connected') {
      throw new BadRequestError('Integration is not connected');
    }

    const accessToken = await this.getValidAccessToken(userId);
    const mappings = this.calendarIntegrationRepo.listByIntegrationId(integration.id);

    let totalPushed = 0;
    let totalPulled = 0;

    for (const mapping of mappings) {
      if (calendarId && mapping.calendar_id !== calendarId) {
        continue;
      }
      if (!mapping.enabled) {
        continue;
      }

      // Check effective sync direction
      const shouldPush =
        (direction === 'push' || direction === 'both') &&
        (mapping.sync_direction === 'push' || mapping.sync_direction === 'bidirectional');
      const shouldPull =
        (direction === 'pull' || direction === 'both') &&
        (mapping.sync_direction === 'pull' || mapping.sync_direction === 'bidirectional');

      // 1. PUSH Lich -> Google
      if (shouldPush) {
        const events = this.eventRepo.findByCalendarId(mapping.calendar_id);
        for (const evt of events) {
          const extMapping = this.eventIntegrationRepo.findByEventAndIntegration(evt.id, integration.id);
          if (extMapping) {
            // Update existing event on Google
            try {
              const updatedExt = await this.provider.updateEvent(
                accessToken,
                mapping.external_calendar_id,
                extMapping.external_id,
                evt,
              );
              this.eventIntegrationRepo.upsert({
                id: extMapping.id,
                event_id: evt.id,
                integration_id: integration.id,
                external_id: updatedExt.id,
                external_updated_at: updatedExt.updated,
              });
              totalPushed++;
            } catch (err: any) {
              // Non-fatal per event
            }
          } else {
            // Create on Google
            try {
              const createdExt = await this.provider.createEvent(
                accessToken,
                mapping.external_calendar_id,
                evt,
              );
              this.eventIntegrationRepo.upsert({
                id: crypto.randomUUID(),
                event_id: evt.id,
                integration_id: integration.id,
                external_id: createdExt.id,
                external_updated_at: createdExt.updated,
              });
              totalPushed++;
            } catch (err: any) {
              // Non-fatal per event
            }
          }
        }
      }

      // 2. PULL Google -> Lich
      if (shouldPull) {
        const syncResource = `calendar:${mapping.external_calendar_id}`;
        const syncState = this.calendarIntegrationRepo.getSyncState(integration.id, syncResource);
        const eventsRes = await this.provider.listEvents(
          accessToken,
          mapping.external_calendar_id,
          syncState?.cursor,
        );

        for (const gEvent of eventsRes.items) {
          const extMapping = this.eventIntegrationRepo.findByExternalId(integration.id, gEvent.id);

          if (gEvent.status === 'cancelled') {
            // Deleted on Google
            if (extMapping) {
              this.eventRepo.softDelete(extMapping.event_id);
              this.eventIntegrationRepo.deleteByEventAndIntegration(extMapping.event_id, integration.id);
              totalPulled++;
            }
            continue;
          }

          const lichFields = GoogleEventMapper.toLich(gEvent, mapping.calendar_id);

          if (extMapping) {
            // Update local event
            const existingLocal = this.eventRepo.findById(extMapping.event_id);
            if (existingLocal) {
              this.eventRepo.update(existingLocal.id, lichFields);
              this.eventIntegrationRepo.upsert({
                id: extMapping.id,
                event_id: existingLocal.id,
                integration_id: integration.id,
                external_id: gEvent.id,
                external_updated_at: gEvent.updated,
              });
              totalPulled++;
            }
          } else {
            // Create new local event
            const newEventId = crypto.randomUUID();
            const created = this.eventRepo.create({
              id: newEventId,
              calendar_id: mapping.calendar_id,
              ...lichFields,
            });

            this.eventIntegrationRepo.upsert({
              id: crypto.randomUUID(),
              event_id: created.id,
              integration_id: integration.id,
              external_id: gEvent.id,
              external_updated_at: gEvent.updated,
            });
            totalPulled++;
          }
        }

        // Save syncToken cursor
        if (eventsRes.nextSyncToken) {
          this.calendarIntegrationRepo.setSyncState(
            crypto.randomUUID(),
            integration.id,
            syncResource,
            eventsRes.nextSyncToken,
          );
        }
      }
    }

    return { pushed: totalPushed, pulled: totalPulled };
  }

  public disconnect(userId: string): void {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (integration) {
      this.integrationRepo.delete(integration.id);
    }
  }

  private async getValidAccessToken(userId: string): Promise<string> {
    const integration = this.integrationRepo.findByUserAndProvider(userId, this.provider.name);
    if (!integration || integration.status !== 'connected') {
      throw new BadRequestError('Integration is not connected');
    }

    const creds = this.integrationRepo.getCredentials(integration.id);
    if (!creds) {
      throw new BadRequestError('No credentials found for integration');
    }

    // Refresh token if expired
    if (creds.expires_at && new Date(creds.expires_at).getTime() < Date.now() + 60000) {
      if (creds.refresh_token) {
        try {
          const refreshed = await this.provider.refreshToken(creds.refresh_token);
          const updated = this.integrationRepo.saveCredentials({
            integration_id: integration.id,
            access_token: refreshed.accessToken,
            refresh_token: refreshed.refreshToken || creds.refresh_token,
            expires_at: refreshed.expiresIn
              ? new Date(Date.now() + refreshed.expiresIn * 1000).toISOString()
              : null,
          });
          return updated.access_token;
        } catch {
          this.integrationRepo.updateStatus(integration.id, 'needs_reauth');
          throw new BadRequestError('Integration token expired and failed to refresh. Please reconnect');
        }
      }
    }

    return creds.access_token;
  }
}
