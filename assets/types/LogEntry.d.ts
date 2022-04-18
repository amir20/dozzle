export interface LogEntry {
  date: Date;
  message: string;
  key: string;
  event?: string;
  selected?: boolean;
}
