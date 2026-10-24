"""Inspect Python function documentation without importing or running scripts."""

import ast
import json
from pathlib import Path


def inventory():
    """Return named Python functions, their docstrings and source locations."""
    out = []
    for path in sorted(Path("scripts").glob("*.py")):
        tree = ast.parse(path.read_text(encoding="utf-8-sig"), filename=str(path))
        for node in ast.walk(tree):
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                out.append({"path": path.as_posix(), "name": node.name,
                            "line": node.lineno, "end": node.end_lineno,
                            "comment": ast.get_docstring(node) or "",
                            "kind": "function"})
    return out


if __name__ == "__main__":
    print(json.dumps(inventory(), ensure_ascii=True))
