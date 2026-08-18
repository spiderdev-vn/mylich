export interface SyncChangeItem {
  id: string;
  entity_type: 'event' | 'calendar';
  entity_id: string;
  action: 'create' | 'update' | 'delete';
  data: any | null;
  created_at: string;
}

export interface SyncResponse {
  cursor: string;
  changes: SyncChangeItem[];
}
