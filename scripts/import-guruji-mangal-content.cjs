const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const zlib = require("node:zlib");
const assert = require("node:assert/strict");

/** importContent archives the supplied bytes and indexes all 24 numbered Hindi rules without rewriting the source. */
function importContent(source) {
  assert.ok(source, "Pass the path to Guruji's supplied text file.");
  const bytes = fs.readFileSync(source);
  const original = bytes.toString("utf8");
  const chunks = original.split("⸻");
  assert.equal(chunks.length, 24, "Expected the 24 source sections.");
  const firstHeading = /(?:^|\r?\n)(1\. [^\r\n]+)/.exec(chunks[0]);
  assert.ok(firstHeading, "First numbered heading missing.");
  const intro = chunks[0].slice(0, firstHeading.index).trim();
  const rules = chunks.map((chunk, index) => {
    const text = index === 0 ? chunk.slice(firstHeading.index).trim() : chunk.trim();
    const newline = text.indexOf("\n");
    const heading = text.slice(0, newline).trim();
    assert.ok(heading.startsWith(`${index + 1}. `), `Unexpected section ${index + 1}`);
    return { number: index + 1, heading, body: text.slice(newline + 1).trim() };
  });
  const archive = "references/guruji/2026-11-25/mangal-parihar";
  fs.mkdirSync(archive, { recursive: true });
  const archivedPath = path.join(archive, "guruji-mangal-parihar.txt");
  if (fs.existsSync(archivedPath)) assert.deepEqual(fs.readFileSync(archivedPath), bytes, "Existing archive differs.");
  else fs.writeFileSync(archivedPath, bytes, { flag: "wx" });
  const sourceInfo = { file: archivedPath.replaceAll("\\", "/"), bytes: bytes.length, sha256: crypto.createHash("sha256").update(bytes).digest("hex"), language: "hi" };
  const catalog = { source: sourceInfo, original_text: original, introduction: intro, rules };
  const json = JSON.stringify(catalog);
  fs.writeFileSync("internal/swisseph/data/guruji-mangal-parihar.json", json + "\n");
  fs.writeFileSync(path.join(archive, "manifest.json"), JSON.stringify(sourceInfo, null, 2) + "\n");
  const output = "output/guruji-bhagyaphal";
  fs.mkdirSync(output, { recursive: true });
  fs.writeFileSync(path.join(output, "guruji-mangal-parihar-hi.min.json"), json);
  fs.writeFileSync(path.join(output, "guruji-mangal-parihar-hi.min.json.gz"), zlib.gzipSync(json));
  assert.equal(JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(output, "guruji-mangal-parihar-hi.min.json.gz"))).toString()).original_text, original);
  console.log(JSON.stringify({ rules: rules.length, archivedBytes: bytes.length, sourceHash: sourceInfo.sha256, originalTextPreserved: true }));
}

importContent(process.argv[2]);
