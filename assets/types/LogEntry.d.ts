export interface LogEntry {
  date: Date;
  message: string;
  key: string;
  event?: string;
  selected?: boolean;
}

export interface LogEvent {
  m?: string;
  ts: number;
  d?: Object;
}
