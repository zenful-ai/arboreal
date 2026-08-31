#!/usr/bin/env sh
# Verifies the book: every included example compiles, every {{#include}}
# points at a real file and a real ANCHOR marker, and mdbook builds without
# a single warning or error.
set -eu
cd "$(dirname "$0")/.."

go build ./...
# engine/ has pre-existing vet failures unrelated to the book; vet only what the book includes.
go vet . ./examples/... ./book/examples/...

# mdbook silently emits an empty code block for an include whose anchor does
# not exist, so check every include target at the source before building.
grep -rHoE --include='*.md' '\{\{#include [^}]+\}\}' book/src | while IFS= read -r line; do
  chapter=${line%%:*}
  spec=${line#*:}
  spec=${spec#\{\{#include }
  spec=${spec%\}\}}
  path=${spec%%:*}
  anchor=${spec#"$path"}
  anchor=${anchor#:}
  file="$(dirname "$chapter")/$path"
  if [ ! -f "$file" ]; then
    echo "book/check.sh: $chapter includes missing file $path" >&2
    exit 1
  fi
  if [ -n "$anchor" ] && ! grep -Eq "ANCHOR: ${anchor}([^A-Za-z0-9_-]|\$)" "$file"; then
    echo "book/check.sh: $chapter includes anchor '$anchor' not found in $path" >&2
    exit 1
  fi
done

log=$(mdbook build book 2>&1) || { printf '%s\n' "$log"; exit 1; }
# mdbook 0.5 logs right-aligned level words without brackets: " WARN ..." / "ERROR ...".
if printf '%s\n' "$log" | grep -Eq '(^|[[:space:]])(WARN|ERROR)[[:space:]]'; then
  printf '%s\n' "$log"
  echo "book/check.sh: mdbook build produced warnings or errors" >&2
  exit 1
fi

# Only scan the HTML mdbook produced; stray editor files (e.g. Emacs
# .~undo-tree~ droppings) are copied into book/book/ verbatim and must not
# trip this check.
if grep -rq --include='*.html' '{{#include' book/book/; then
  echo "book/check.sh: an include directive survived into the built HTML" >&2
  grep -rn --include='*.html' '{{#include' book/book/ >&2
  exit 1
fi

echo "book OK"
