export interface LogEntry {
  readonly date: Date;
  readonly message: string | JSONObject;
  readonly id: number;
  event?: string;
  selected?: boolean;
}

export interface LogEvent {
  readonly m: string | JSONObject;
  readonly ts: number;
  readonly id: number;
}

export type JSONValue = string | number | boolean | JSONObject | Array<JSONValue>;
export type JSONObject = { [x: string]: JSONValue };
