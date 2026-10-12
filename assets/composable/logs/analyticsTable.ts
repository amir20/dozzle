import type { Container } from "@/models/Container";
import { appendContainerRefs } from "./containerRefs";

/** Every source's JSON lines in one response, one container after another. */
export function analyticsLogsParams(containers: Container[]): URLSearchParams {
  const params = new URLSearchParams({ format: "jsonl", stdout: "1", stderr: "1" });
  appendContainerRefs(params, containers);
  return params;
}

/**
 * The statement that turns the fetched lines in `file` into the `logs` table. Several
 * containers add `_container` (from each event's `c`) and `_ts`, underscored so they
 * rarely collide with a field the app logs itself (`ts` often does). They also infer the
 * schema from every line: the default sample stops inside the first container, and the
 * fields only a later one logs would be dropped.
 */
export function createLogsTableSql(file: string, merged: boolean): string {
  const options = `ignore_errors = true, format = 'newline_delimited', map_inference_threshold = -1`;
  if (!merged) return `CREATE TABLE logs AS SELECT unnest(m) FROM read_json('${file}', ${options})`;
  return `CREATE TABLE logs AS SELECT c AS _container, epoch_ms(ts) AS _ts, unnest(m) FROM read_json('${file}', ${options}, sample_size = -1)`;
}
