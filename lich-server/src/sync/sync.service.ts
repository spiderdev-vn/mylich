import type { ChangeLogRepository } from '../db/repositories/change_log.repository.ts';
import type { SyncResponse, SyncChangeItem } from './sync.types.ts';

export class SyncService {
  private changeLogRepo: ChangeLogRepository;

  constructor(changeLogRepo: ChangeLogRepository) {
    this.changeLogRepo = changeLogRepo;
  }

  public getSyncData(userId: string, since?: string, limit: number = 100): SyncResponse {
    const records = this.changeLogRepo.getChangesSince(userId, since, limit);

    const changes: SyncChangeItem[] = records.map((r) => ({
      id: r.id,
      entity_type: r.entity_type,
      entity_id: r.entity_id,
      action: r.action,
      data: r.data ? JSON.parse(r.data) : null,
      created_at: r.created_at,
    }));

    // Next cursor is the latest change's created_at, or the current server time if empty
    let cursor = new Date().toISOString();
    if (changes.length > 0) {
      cursor = changes[changes.length - 1].created_at;
    } else if (since) {
      cursor = since;
    }

    return {
      cursor,
      changes,
    };
  }
}
