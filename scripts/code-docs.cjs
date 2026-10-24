// Build and verify the first-party function index without executing app code.
const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const { createRequire } = require("node:module");
const root = path.resolve(__dirname, "..");
const requireWeb = createRequire(path.join(root, "web/package.json"));
const { parse } = requireWeb("@babel/parser");

/** filesIn lists maintained source files while excluding generated artifacts. */
function filesIn(directory) {
  return fs.readdirSync(path.join(root, directory), { withFileTypes: true })
    .flatMap((entry) => {
      const file = `${directory}/${entry.name}`;
      return entry.isDirectory() ? filesIn(file) : [file];
    });
}

/** attachedComment collects the contiguous documentation immediately above a declaration. */
function attachedComment(ast, anchor, code) {
  const comments = ast.comments.filter((comment) => comment.end <= anchor.start);
  const selected = [];
  let boundary = anchor.start, line = anchor.loc.start.line;
  for (let index = comments.length - 1; index >= 0; index--) {
    const comment = comments[index];
    if (comment.loc.end.line !== line - 1 || code.slice(comment.end, boundary).trim()) break;
    selected.unshift(comment);
    boundary = comment.start;
    line = comment.loc.start.line;
  }
  return selected.map((comment) => comment.value.replace(/^\s*\* ?/gm, "").trim()).join("\n");
}

/** javascriptSymbols parses named functions, React components, closures and methods. */
function javascriptSymbols(file) {
  const code = fs.readFileSync(path.join(root, file), "utf8");
  const ast = parse(code, {
    sourceType: "unambiguous",
    plugins: file.endsWith(".cjs") ? [] : ["typescript", "jsx"],
  });
  const out = [];
  /** visit walks syntax nodes and resolves a stable name for each documented unit. */
  function visit(node, ancestors = []) {
    if (!node || typeof node !== "object" || !node.type) return;
    const parent = ancestors.at(-1);
    let name, anchor = node, kind = "function";
    if (node.type === "FunctionDeclaration" && node.id) {
      name = node.id.name;
      if (parent?.type.startsWith("Export")) anchor = parent;
    } else if (["ArrowFunctionExpression", "FunctionExpression"].includes(node.type)) {
      let owner = parent;
      if (owner?.type === "CallExpression" && ["useCallback", "useMemo", "memo", "forwardRef"].includes(owner.callee.name)) {
        owner = ancestors.at(-2);
      }
      if (owner?.type === "VariableDeclarator" && owner.id.type === "Identifier") {
        name = owner.id.name;
        anchor = ancestors.findLast((item) => item.type === "VariableDeclaration") || owner;
        const exportNode = ancestors.findLast((item) => item.type.startsWith("Export"));
        if (exportNode && exportNode.declaration === anchor) anchor = exportNode;
        kind = "closure";
      } else if (owner?.type === "ObjectProperty" && owner.key.type === "Identifier") {
        name = owner.key.name;
        anchor = owner;
        kind = "method";
      }
    } else if (["ClassMethod", "ObjectMethod"].includes(node.type)) {
      name = node.key.name || String(node.key.value);
      kind = "method";
      const owner = ancestors.findLast((item) => item.type === "ClassDeclaration");
      if (owner?.id) name = `${owner.id.name}.${name}`;
    }
    if (name) out.push({ path: file, name, line: anchor.loc.start.line,
      end: node.loc.end.line, comment: attachedComment(ast, anchor, code), kind });
    for (const [key, child] of Object.entries(node)) {
      if (["loc", "comments", "leadingComments", "trailingComments", "innerComments"].includes(key)) continue;
      if (Array.isArray(child)) child.forEach((item) => visit(item, [...ancestors, node]));
      else if (child && typeof child === "object") visit(child, [...ancestors, node]);
    }
  }
  visit(ast.program);
  return out;
}

/** inventory combines Go, browser/Node and Python declarations using language parsers. */
function inventory() {
  const options = { cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    env: { ...process.env, GOMAXPROCS: "2" }, windowsHide: true };
  const go = JSON.parse(execFileSync("go", ["run", "-p", "1", "./cmd/codemap"], options));
  const python = JSON.parse(execFileSync("python", ["scripts/code_docs_python.py"], options));
  const jsFiles = ["web/src", "web/tests", "scripts"].flatMap(filesIn)
    .filter((file) => /\.(?:tsx?|cjs)$/.test(file));
  const js = jsFiles.flatMap(javascriptSymbols);
  return [...go, ...js, ...python].sort((a, b) =>
    a.path.localeCompare(b.path) || a.line - b.line || a.name.localeCompare(b.name));
}

/** markdown renders the searchable function directory from current source comments. */
function markdown(symbols) {
  const groups = Map.groupBy(symbols, (symbol) => symbol.path);
  const lines = ["# Function index", "", "Generated from source comments. Run `node scripts/code-docs.cjs --write` after changing functions or comments.", "",
    "Start with [the codebase map](CODEBASE.md) for feature ownership and request flows.", "",
    `This index covers ${symbols.length} named functions, methods and local helpers in ${groups.size} first-party files. Inline callbacks belong to their enclosing documented function; generated output and the bundled Swiss C library are excluded.`, ""];
  for (const [file, functions] of groups) {
    lines.push(`## ${file}`, "", "| Function | Purpose |", "|---|---|");
    for (const symbol of functions) {
      const purpose = symbol.comment.split(/\n\s*\n/)[0].replace(/\n/g, " ").replace(/\|/g, "\\|");
      lines.push(`| [\`${symbol.name}\`](../${file}#L${symbol.line}) | ${purpose || "Missing documentation."} |`);
    }
    lines.push("");
  }
  return lines.join("\n");
}

/** main reports missing comments or refreshes the checked-in function index. */
function main() {
  const symbols = inventory();
  if (process.argv.includes("--inventory")) {
    process.stdout.write(JSON.stringify(symbols, null, 2));
    return;
  }
  const missing = symbols.filter((symbol) => !symbol.comment.trim());
  if (missing.length) {
    for (const symbol of missing) console.error(`${symbol.path}:${symbol.line} ${symbol.name}: add a purpose comment`);
    console.error(`${missing.length} of ${symbols.length} named functions need documentation.`);
    process.exitCode = 1;
    return;
  }
  const rendered = markdown(symbols);
  const destination = path.join(root, "docs/FUNCTION_INDEX.md");
  if (process.argv.includes("--write")) fs.writeFileSync(destination, rendered);
  else if (!fs.existsSync(destination) || fs.readFileSync(destination, "utf8") !== rendered) {
    console.error("Function index is stale. Run: node scripts/code-docs.cjs --write");
    process.exitCode = 1;
    return;
  }
  console.log(`Documentation verified: ${symbols.length} named functions; current function index.`);
}

main();
