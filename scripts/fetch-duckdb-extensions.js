// Vendors the DuckDB extensions SQL Analytics needs into public/, so the browser loads
// them from Dozzle instead of extensions.duckdb.org.
//
// Only core_functions is statically linked into duckdb-wasm. `read_json` autoloads the
// json extension from a repository laid out as <repo>/<version>/<platform>/<name>, and
// useDuckDB points custom_extension_repository at public/duckdb-extensions. The
// extension has to match the engine exactly, so the version is asked of the engine
// itself rather than pinned here, where a dependency bump would silently break it.
//
// Only wasm_eh is fetched: useDuckDB always loads the eh bundle.
import { existsSync, mkdirSync, readdirSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { join } from "node:path";
import * as duckdb from "@duckdb/duckdb-wasm/dist/duckdb-node-blocking.cjs";

const OUT = "public/duckdb-extensions";
const PLATFORM = "wasm_eh";
const EXTENSIONS = ["json"];

// The wasm also embeds a table of every storage version it can read, so grepping it
// for a version string finds dozens. Asking the engine takes under a second.
const wasm = createRequire(import.meta.url).resolve("@duckdb/duckdb-wasm/dist/duckdb-eh.wasm");
const bundle = { mainModule: wasm, mainWorker: "" };
const db = await duckdb.createDuckDB({ mvp: bundle, eh: bundle }, new duckdb.VoidLogger(), duckdb.NODE_RUNTIME);
await db.instantiate();
db.open({});
const version = db.connect().query("SELECT library_version FROM pragma_version()").toArray()[0].library_version;
if (!/^v\d+\.\d+\.\d+$/.test(version)) throw new Error(`unexpected DuckDB version: ${version}`);

// A stale version left in public/ would be copied into dist and embedded for nothing.
if (existsSync(OUT)) {
  for (const entry of readdirSync(OUT)) {
    if (entry !== version) rmSync(join(OUT, entry), { recursive: true });
  }
}

const dir = join(OUT, version, PLATFORM);
mkdirSync(dir, { recursive: true });

for (const name of EXTENSIONS) {
  const file = join(dir, `${name}.duckdb_extension.wasm`);
  if (existsSync(file)) continue;

  const url = `https://extensions.duckdb.org/${version}/${PLATFORM}/${name}.duckdb_extension.wasm`;
  const response = await fetch(url);
  if (!response.ok) throw new Error(`${url}: ${response.status} ${response.statusText}`);
  // Written aside and renamed, so an interrupted build cannot leave a truncated file
  // that the existsSync above would then trust forever. A leftover .part is simply
  // overwritten by the next run.
  const partial = `${file}.part`;
  writeFileSync(partial, new Uint8Array(await response.arrayBuffer()));
  renameSync(partial, file);
  console.log(`duckdb: fetched ${name} ${version}`);
}
