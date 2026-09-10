import * as duckdb from "@duckdb/duckdb-wasm";
const JSDELIVR_BUNDLES = duckdb.getJsDelivrBundles();

export async function useDuckDB() {
  let cleanup: (() => void) | undefined;
  onUnmounted(() => cleanup?.());

  const bundle = await duckdb.selectBundle(JSDELIVR_BUNDLES);
  const worker_url = URL.createObjectURL(
    new Blob([`importScripts("${bundle.mainWorker!}");`], { type: "text/javascript" }),
  );

  // Instantiate the asynchronus version of DuckDB-Wasm
  const worker = new Worker(worker_url);
  const logger = new duckdb.ConsoleLogger();
  const db = new duckdb.AsyncDuckDB(logger, worker);

  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  URL.revokeObjectURL(worker_url);
  const conn = await db.connect();

  cleanup = async () => {
    console.log("Cleaning up DuckDB");
    await conn.close();
    await db.terminate();
    worker.terminate();
  };

  return { db, conn };
}
