import type { AsyncDuckDB, AsyncDuckDBConnection } from "@duckdb/duckdb-wasm";
import ehWasm from "@duckdb/duckdb-wasm/dist/duckdb-eh.wasm?url";
import ehWorker from "@duckdb/duckdb-wasm/dist/duckdb-browser-eh.worker.js?url";

// Uncompressed size of duckdb-eh.wasm, injected by vite.config.ts. DuckDB reports
// progress as decoded bytes against the response's Content-Length, which is the
// brotli size, so its own ratio runs to about 660%.
declare const __DUCKDB_WASM_SIZE__: number;

// Served by Dozzle rather than a CDN, so SQL Analytics works offline and tells no one
// it was opened. Only the eh build ships: every browser Dozzle supports has wasm
// exceptions, and a second build would add another ~6 MB to the binary.
const bundle = {
  mainModule: new URL(ehWasm, import.meta.url).href,
  mainWorker: new URL(ehWorker, import.meta.url).href,
};

/**
 * Starts DuckDB and resolves once it can take queries. Call it synchronously in setup:
 * it registers its own cleanup, and the component may unmount before startup finishes.
 * `onProgress` receives the engine download as a fraction from 0 to 1.
 */
export function useDuckDB(onProgress?: (fraction: number) => void) {
  let unmounted = false;
  let cleanup: (() => Promise<void>) | undefined;
  onUnmounted(() => {
    unmounted = true;
    cleanup?.().catch((e) => console.warn("DuckDB cleanup failed", e));
  });

  return start();

  async function start(): Promise<{ db: AsyncDuckDB; conn: AsyncDuckDBConnection }> {
    // The library is only needed once someone opens SQL Analytics.
    const duckdb = await import("@duckdb/duckdb-wasm");

    // A classic worker cannot be constructed from another origin, which is where vite
    // serves it in dev, but it can importScripts from one.
    const worker_url = URL.createObjectURL(
      new Blob([`importScripts("${bundle.mainWorker}");`], { type: "text/javascript" }),
    );

    // Instantiate the asynchronous version of DuckDB-Wasm
    const worker = new Worker(worker_url);
    URL.revokeObjectURL(worker_url);
    const db = new duckdb.AsyncDuckDB(new duckdb.ConsoleLogger(), worker);

    let conn: AsyncDuckDBConnection;
    try {
      await db.instantiate(bundle.mainModule, null, (p) =>
        onProgress?.(Math.min(p.bytesLoaded / __DUCKDB_WASM_SIZE__, 1)),
      );
      onProgress?.(1);
      // Arrow hands DECIMAL back as an unscaled integer, so `latency_ms * 1.5` read ten
      // times too large in both the table and the chart. Doubles are what both render.
      await db.open({ query: { castDecimalToDouble: true } });
      conn = await db.connect();
      // read_json autoloads the json extension, vendored by scripts/fetch-duckdb-extensions.js.
      // Anything not vendored fails to load rather than falling back to extensions.duckdb.org.
      const repository = new URL(withBase("/duckdb-extensions"), location.origin).href;
      await conn.query(`SET custom_extension_repository = '${repository.replaceAll("'", "''")}'`);
    } catch (e) {
      worker.terminate();
      throw e;
    }

    // terminate() also stops the worker.
    cleanup = async () => {
      try {
        await conn.close();
      } finally {
        await db.terminate();
      }
    };
    if (unmounted) {
      await cleanup();
      // Nothing is left to hand the connection to, and rejecting would log an error for
      // a drawer the user simply closed, so the caller is left pending instead.
      return new Promise<never>(() => {});
    }

    return { db, conn };
  }
}
