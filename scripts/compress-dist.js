// Precompresses the built assets with brotli and drops the originals.
//
// The server used to brotli every static response on the fly, at the library's
// default quality, for every visitor. Compressing once at quality 11 is both
// smaller and free at request time. The uncompressed copies are removed rather
// than kept alongside, so the embedded FS (and the binary) shrinks instead of
// doubling; internal/web/index.go serves the `.br` sibling and inflates it for
// the rare client that does not send `Accept-Encoding: br`.
//
// index.html and .vite/manifest.json are left alone: Go reads and parses both.
import { constants, brotliCompressSync } from "node:zlib";
import { readdirSync, readFileSync, writeFileSync, unlinkSync, statSync } from "node:fs";
import { join, extname } from "node:path";

const DIST = "dist";
const COMPRESSIBLE = new Set([".js", ".css", ".svg", ".json", ".map"]);
const SKIP = new Set([join(DIST, "index.html"), join(DIST, ".vite", "manifest.json")]);

function* walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) yield* walk(path);
    else yield path;
  }
}

let before = 0;
let after = 0;
let count = 0;

for (const file of walk(DIST)) {
  const size = statSync(file).size;
  before += size;

  if (SKIP.has(file) || !COMPRESSIBLE.has(extname(file))) {
    after += size;
    continue;
  }

  const compressed = brotliCompressSync(readFileSync(file), {
    params: {
      [constants.BROTLI_PARAM_QUALITY]: constants.BROTLI_MAX_QUALITY,
      [constants.BROTLI_PARAM_SIZE_HINT]: size,
    },
  });

  // Brotli can lose on tiny files. Keeping the original also keeps the server's
  // fast path honest, so only swap when it actually pays.
  if (compressed.length >= size) {
    after += size;
    continue;
  }

  writeFileSync(`${file}.br`, compressed);
  unlinkSync(file);
  after += compressed.length;
  count++;
}

const mb = (n) => (n / 1024 / 1024).toFixed(2);
console.log(`brotli: ${count} files, dist ${mb(before)} MB -> ${mb(after)} MB`);
