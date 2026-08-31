# The Arboreal Book

An mdbook documenting the Arboreal framework for engineers coming from
LangGraph. Sources live in `src/`; every code listing is `{{#include}}`d
from a compiled package under `../examples/`, never pasted in.

## Building

From the repository root:

```sh
mdbook build book    # output in book/book/
mdbook serve book    # live preview at http://localhost:3000
```

The gate is `book/check.sh`, run it before committing book changes:

```sh
./book/check.sh
```

It compiles every included example (`go build ./...`, `go vet . ./examples/...`),
verifies each `{{#include}}` path and `ANCHOR:` marker at the source, requires a
warning-free `mdbook build`, and confirms no include directive leaked into the
built HTML. It prints `book OK` on success.

## Dependencies

| Tool            | Version                    | Install                               |
|-----------------|----------------------------|---------------------------------------|
| Go toolchain    | whatever builds the repo   | needed by `check.sh` for the examples |
| mdbook          | 0.5.x (built with 0.5.4)   | `cargo install mdbook`                |
| mdbook-mermaid  | 0.17.1+                    | `cargo install mdbook-mermaid`        |
| mdbook-svgbob   | 0.3.1+                     | `cargo install mdbook-svgbob`         |
| mdbook-admonish | **fork build — see below** | see below                             |

### mdbook-admonish comes from a fork

The released mdbook-admonish (1.20.0, crates.io) is incompatible with
mdbook 0.5: the 0.5 preprocessor protocol renamed the book's `sections`
field to `items`, and serializes an unset `text-direction` as JSON `null`,
which the plugin's TOML-based config deserializer rejects. The fix is in
[PR #235](https://github.com/tommilligan/mdbook-admonish/pull/235)
(tracking issue [#233](https://github.com/tommilligan/mdbook-admonish/issues/233)).
Until it merges, install from the PR branch:

```sh
cargo install --git https://github.com/padamson/mdbook-admonish \
  --branch feat/mdbook-0.5-compat mdbook-admonish
```

Note: the fork still reports `1.20.0` from `--version`, so the version
string cannot tell you which build you have. If `mdbook build book` fails
with `missing field 'sections'`, you have the crates.io release — reinstall
from the fork. Once the PR merges upstream, switch back with
`cargo install mdbook-admonish --force`.

Two related pins in `book.toml`, neither to be removed:

- `text-direction = "ltr"` under `[book]` — works around the JSON-null
  issue described above.
- `assets_version = "3.1.0"` under `[preprocessor.admonish]` — managed by
  `mdbook-admonish install`; do not edit by hand.
