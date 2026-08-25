#!/usr/bin/env python3
"""Works around a known tfplugingen-framework bug (hashicorp/terraform-plugin-codegen-framework
issues #152/#156/#236) where nested-object attribute types get emitted twice, verbatim,
in the same _gen.go file, causing "redeclared in this block" compile errors.

Splits each generated file into top-level declarations (type/func/var/const starting at
column 0) and drops any declaration whose full text exactly repeats one already emitted.
Safe because the duplicates are confirmed byte-identical, not divergent overloads.

Run after `tfplugingen-framework generate all`, before `gofmt`/`go build`.
"""
import re
import sys
from pathlib import Path

GEN_ROOT = Path(__file__).parent.parent / "internal" / "provider" / "generated"
DECL_START = re.compile(r"^(type|func|var|const)\s")


def dedupe_file(path):
    lines = path.read_text().splitlines(keepends=True)
    header = []
    decls = []
    current = None
    in_header = True
    for line in lines:
        if DECL_START.match(line):
            in_header = False
            if current is not None:
                decls.append(current)
            current = [line]
        elif in_header:
            header.append(line)
        else:
            current.append(line)
    if current is not None:
        decls.append(current)

    seen = set()
    kept = []
    removed = 0
    for decl in decls:
        # Compare ignoring trailing blank lines: the generator sometimes emits
        # an otherwise-identical duplicate decl with one fewer/more trailing
        # newline than the original, which byte-exact comparison would miss.
        key = "".join(decl).rstrip("\n")
        if key in seen:
            removed += 1
            continue
        seen.add(key)
        kept.append(decl)

    if removed:
        path.write_text("".join(header) + "".join("".join(d) for d in kept))
    return removed


def main():
    total = 0
    files = sorted(GEN_ROOT.glob("**/*_gen.go"))
    for f in files:
        removed = dedupe_file(f)
        if removed:
            print(f"{f.relative_to(GEN_ROOT.parent.parent.parent)}: removed {removed} duplicate declaration(s)")
            total += removed
    print(f"total duplicates removed: {total}")


if __name__ == "__main__":
    main()
