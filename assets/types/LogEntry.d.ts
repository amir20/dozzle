export interface LogEntry {
  readonly date: Date;
  readonly message?: string;
  readonly payload?: Record<string, any>;
  readonly id: number;
  event?: string;
  selected?: boolean;
}

export interface LogEvent {
  readonly m?: string;
  readonly ts: number;
  readonly d?: Record<string, any>;
  readonly id: number;
}
