export interface LogEntry {
  date: Date;
  message?: string;
  payload?: Record<string, any>;
  key: string;
  event?: string;
  selected?: boolean;
}

export interface LogEvent {
  m?: string;
  ts: number;
  d?: Record<string, any>;
}
