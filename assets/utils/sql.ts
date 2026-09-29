const quote = (name: string) => `"${name.replaceAll('"', '""')}"`;

/**
 * The query SQL Analytics opens with when a numeric field is charted from LogDetails.
 * `logs` is built by unnesting each line's JSON, so a top-level key is a column and a
 * nested one is a struct field reached through it. TRY_CAST keeps the rows where the
 * same key held text from failing the whole query.
 */
export function numericFieldQuery(key: string[]): string {
  const path = key.length > 1 ? ["logs", ...key].map(quote).join(".") : quote(key[0]);
  const value = `TRY_CAST(${path} AS DOUBLE)`;
  return `SELECT ${value} AS ${quote(key.join("."))}\nFROM logs\nWHERE ${value} IS NOT NULL`;
}
