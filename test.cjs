const duckdb = require("@duckdb/duckdb-wasm");
const Arrow = require("apache-arrow");
const path = require("path");
const Worker = require("web-worker");
const DUCKDB_DIST = path.dirname(require.resolve("@duckdb/duckdb-wasm"));

(async () => {
  try {
    const DUCKDB_CONFIG = await duckdb.selectBundle({
      mvp: {
        mainModule: path.resolve(DUCKDB_DIST, "./duckdb-mvp.wasm"),
        mainWorker: path.resolve(DUCKDB_DIST, "./duckdb-node-mvp.worker.cjs"),
      },
      eh: {
        mainModule: path.resolve(DUCKDB_DIST, "./duckdb-eh.wasm"),
        mainWorker: path.resolve(DUCKDB_DIST, "./duckdb-node-eh.worker.cjs"),
      },
    });

    const logger = new duckdb.ConsoleLogger();
    const worker = new Worker(DUCKDB_CONFIG.mainWorker);
    const db = new duckdb.AsyncDuckDB(logger, worker);
    await db.instantiate(DUCKDB_CONFIG.mainModule, DUCKDB_CONFIG.pthreadWorker);

    // await db.registerFileURL(
    //   "logs.json",
    //    "http://localhost:3100/api/hosts/ivkagb8ir869qgj2ft73t2fbg/containers/9da1a8d03a8a/logs?stdout=1&stderr=1",
    //   "http",
    //   false,
    // );

    const response = await fetch("http://192.168.68.66:8080/logs.json");

    if (!response.ok) {
      throw new Error(`Failed to fetch logs: ${response.statusText}`);
    }

    await db.registerFileBuffer("logs.json", new Uint8Array(await response.arrayBuffer()));

    const conn = await db.connect();

    await conn.insertJSONFromPath("logs.json", { name: "logs", columns: { m: new Arrow.JSONReader() } });
    const results = await conn.query(`
      SELECT * FROM logs.json
      `);

    for (const row of results.toArray()) {
      console.log(row.m.time);
    }

    await conn.close();
    await db.terminate();
    await worker.terminate();
  } catch (e) {
    console.error(e);
  }
})();
