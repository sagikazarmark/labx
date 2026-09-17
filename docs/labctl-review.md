# iximiuz Labs / labctl compatibility review

Researched 2026-09-17 against official documentation and tagged upstream source.

## Summary

- **Latest release: [labctl v0.1.112](https://github.com/iximiuz/labctl/releases/tag/v0.1.112)**, published September 7, 2026. Its release notes include shell-gym support. GitHub's [comparison with main](https://github.com/iximiuz/labctl/compare/v0.1.112...main) reported identical commits at research time.
- Updated labx from **v0.1.72 to v0.1.112** in [go.mod](../go.mod). **Updating the dependency alone does not add play variables or close most content-schema gaps:** labx owns its content manifests, task types, explicit conversions, and Markdown templating.

## Changes implemented

- Added `vars` to core and extended content manifests and their conversion, preserving literal/expression strings and shell/machine objects.
- Documented Go-template literal escaping for native Labs bindings in the [README](../README.md#play-variables). Direct unescaped `{{ vars.NAME }}` remains incompatible with Go-template parsing. `:code-var` needs no escaping.
- Defined the expression policy: reference final generated task names explicitly; labx only rewrites `needs`. Hyphens in generated task names become underscores.
- Added pass-through for task `helper`, `playgrounds` / `skill-paths` embedding maps, and content `playground.registryAuth`.
- The upgraded upstream tab type preserves `pane` and `target`.
- Added an offline generate/render regression test covering these fields, both variable shapes, runtime expressions, expanded task names, and escaped bindings. `go test ./...`, `go test -short -race -shuffle=on ./...`, `go build ./...`, and `golangci-lint run ./...` pass; no live Labs deployment was tested.

### Follow-up implementation

- Added playground-level startup files and per-file `source`, `extract`, and optional `machines` selectors. Both levels process `fromFile`; content conversion appends the base playground's top-level startup files. Machine processing also works when no named base playground is specified.
- Staged sibling archive folders beside generated content/playground manifests (including course lesson outputs), preserving file modes and `.labctlignore`. Generated reproducible `.tar`, `.tar.gz`, and `.tgz` archives with nested ignore rules; prebuilt archives and remote references remain supported.
- Building archives in labx also covers an upstream scanner mismatch: v0.1.112's [`startupFileSources`](https://github.com/iximiuz/labctl/blob/v0.1.112/cmd/content/archives.go) reads flat YAML playground specs, while labx writes wrapped `playground:` manifests. Content front-matter scanning does understand the wrapper. Staging alone would not make standalone playground archives work.
- Added standalone playground `portForwards` pass-through.
- Added explicit index rendering for `skill-path`, `blog-post`, `shell-gym`, `roadmap`, and `vendor`. Skill paths render sibling `unit-*.md` files or labx's `units/` layout, following the [official sample](https://github.com/iximiuz/labs/tree/main/content-samples/sample-skill-path).
- Added typed skill-path `difficulties` and YAML preservation of additional top-level front-matter fields via `.Manifest.Metadata`. This avoids guessing private or evolving schemas for newer content kinds; semantic validation remains with Labs.
- Added tests for startup-file inheritance, archive content/modes/ignore rules/reproducibility, port-forward values, metadata preservation, and new-kind generation/rendering.

### Additional compatibility pass

- Added content `playground.initTasks`, `initConditions`, and `portForwards`. Local init tasks can depend on base-playground tasks, and content tasks can depend on locally declared init tasks with generated names resolved during expansion.
- Fixed omitted task `machine` / `user` dimensions: one task is emitted with the field omitted, leaving defaults to Labs. Explicitly empty target lists retain their existing zero-task behavior.
- Added typed `shellgym.version`, `shellgym.path`, `shellgym.port`, and `repetition`, preserving omitted values instead of applying defaults. The public [format reference](https://github.com/iximiuz/labs/blob/main/content-samples/claude/rules/shell-gym-format-docs.md) documents these fields; [authoring guidance](https://github.com/iximiuz/labs/blob/main/content-samples/claude/rules/shell-gym-authoring.md) requires matching extraction/tool paths and keeping raw learning-path files uploadable for searchability.
- Added unknown-field preservation to locally owned playground, machine, user, startup-file, and task YAML models, including standalone output. Decoding removes known fields from inline metadata maps so build-only fields cannot leak into generated manifests. YAML merge-key behavior is covered by regression tests.
- Added a [Shell Gym fixture](../labx/testdata/shell-gym) following the [tool's path format](https://github.com/iximiuz/shellgym/blob/main/docs/authoring-guide.md). Tests verify typed settings, a named-base lookup via a local stub, omitted task targets, and byte-for-byte preservation of learning-path files both staged and archived.
- Blog-post support remains index rendering plus generic metadata preservation. The [labctl BlogPost type](https://github.com/iximiuz/labctl/blob/v0.1.112/api/blogposts.go) is an API response model, not a confirmed writable front-matter schema; no blog-only fields have been invented from it.

Verification for this pass: `go test ./...`, `go test -short -race -shuffle=on ./...`, `go build ./...`, `golangci-lint run ./...`, and `git diff --check` passed. Live Shell Gym validation was not performed; the installed local `labctl` executable reports v0.1.101, predating Shell Gym support. The repository dependency remains v0.1.112. A live check needs an updated executable and a target gym, followed by `shellgym validate` / `shellgym solve` in its VM as described in the official authoring workflow.

The gap analysis below records the **pre-change baseline**. Unknown-field preservation covers the locally owned models listed above; nested types imported directly from labctl still follow their upstream typed schemas.

## Play variables: platform contract

The official [Play Variables documentation](https://labs.iximiuz.com/tutorials/sample-tutorial#play-variables) defines a top-level `vars` map with two value shapes:

```yaml
vars:
  PORT: "8080"
  POD_IP: x(.tasks.init_lookup.stdout)
  URL: http://x(.vars.POD_IP):x(.vars.PORT)/
  SESSION_ID:
    shell: head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n'
    machine: dev-machine
```

- String values can be literals or `x(...)` templates referencing other variables and task `stdout`, `stderr`, `exit_code`, `input`, or `status`. Resolution waits for referenced tasks to complete and referenced variables to resolve.
- Shell definitions resolve to trimmed stdout; commands retry until exit status zero. `machine` is optional and defaults to the first playground machine.
- Markdown bindings use `{{ vars.NAME || 'fallback' }}` (single-quoted fallback only). Bindings are not expanded inside inline code/code blocks and are invalid inside table cells. `:code-var{name=NAME default='...'}` provides a copyable inline value.
- Task `run`, `env`, `hintcheck`, and `failcheck` can reference `x(.vars.NAME)`; execution blocks until the variable resolves.
- [`::conditional{var=NAME}`](https://labs.iximiuz.com/tutorials/sample-tutorial#conditional-content) selects a named slot matching the value, otherwise the default slot. A variable referencing a task's status resolves to `completed` after completion, enabling gated content.

### Concrete labx gaps

1. **Declarations disappear from the modeled manifest.** Neither [core.ContentManifest](../core/content.go) nor [extended.ContentManifest and Convert](../extended/content.go) has `Vars` or an unknown-field passthrough. Generation decodes into the extended struct, converts explicitly, and serializes the core struct; render-only decodes directly into core. Both paths need a representation preserving string versus `{shell, machine}` values. See [content processing](../labx/content.go), [YAML handling](../labx/common.go), and [render-only handling](../labx/render.go).
2. **Runtime bindings conflict with build-time templates.** [template.go](../labx/template.go) parses Markdown with Go `text/template`'s default `{{ ... }}` delimiters. A native Labs binding is interpreted as a Go action and fails parsing, rather than surviving for Labs to resolve. Both generate and render use this parser. A literal-emission escape such as ``{{ `{{ vars.PORT || '...' }}` }}`` can preserve an individual binding; first-class support needs a documented escape/pass-through or delimiter strategy. `:code-var` and `::conditional` themselves avoid this delimiter collision, but still require the missing declarations.
3. **Expanded task references need a defined policy.** [convertTasks/currentName](../extended/content.go) renames tasks expanded across machines/users and rewrites `needs`, but leaves script strings unchanged. New variable expressions referring to an unexpanded task name would point at a nonexistent task after expansion. Require explicit generated task names, reject ambiguous references, or implement deliberate reference rewriting. Existing `x(.needs...)` expressions deserve the same check.

Recommendation: implement variables as runtime declarations passed through to Labs, separately from labx's build-time `.Extra`/`.Manifest` data. Add focused round-trip coverage for both value shapes, generated front matter, render-only access, preserved bindings, and expanded task references. A dependency bump is neither sufficient nor inherently required for this locally owned schema feature.

## Other relevant gaps and dependency impact

| Feature / official source | Current labx handling | Does a dependency update alone suffice? |
| --- | --- | --- |
| `helper: true` background tasks ([sample source](https://github.com/iximiuz/labs/blob/main/content-samples/sample-tutorial/index.md), [task docs](https://labs.iximiuz.com/tutorials/sample-tutorial#background-tasks)) | Both local content `Task` structs and their conversion omit `helper`; the intended helper classification is lost. | **No.** Add it to core, extended, and conversion. |
| `playgrounds` and `skill-paths` embedding maps; cards and grids ([tutorial docs](https://labs.iximiuz.com/tutorials/sample-tutorial#how-to-embed-a-grid-of-cards)) | Core has only `challenges`/`tutorials`; extended additionally has its own `courses` expansion. Required reference maps are absent even though MDC body text can pass through. | **No.** Extend the local manifests/conversion. |
| Content `playground.registryAuth` ([tutorial docs](https://labs.iximiuz.com/tutorials/sample-tutorial#playground-container-registry)) | Missing from both local content playground specs. Standalone extended playgrounds already preserve it. | **No.** Wire it through the content path. |
| Playground-level `startupFiles`, optional `machines`, and file `source` / `extract` ([official startup-file docs](https://labs.iximiuz.com/docs/custom-playgrounds/init-tasks#startup-files), [v0.1.112 types][new-api]) | Both extended playground specs and the core content playground spec lack the top-level list. Local extended per-machine startup files lack `source`/`extract`. Existing support is inline/per-machine, plus `fromFile`. | **No** for generation. Updating imported types helps direct core/render paths, but extended fields, conversions, and parent-playground handling still need changes. |
| Tab `pane` / `target` ([tutorial tabs](https://labs.iximiuz.com/tutorials/sample-tutorial#controlling-ui-tabs), [new types][new-api] versus [old types][old-api]) | Specs use `[]api.PlaygroundTab` directly and conversions copy the slice. v0.1.72 lacks these fields; v0.1.112 includes both. | **Yes at the schema/pass-through level**, subject to an upgrade build and round-trip check. |
| Standalone playground `portForwards` ([upstream types][new-api]) | Upstream already has this in v0.1.72, but local `extended.PlaygroundSpec` and its conversion omit it. | **No.** This is local schema drift, not dependency age. |
| Additional kinds, including `skill-path`, `blog-post`, and latest `shell-gym` ([upstream kinds](https://github.com/iximiuz/labctl/blob/v0.1.112/content/content.go)) | Render explicitly handles challenge/tutorial/training/course plus playground. Generate routes all other kinds through generic content handling without kind-specific schema support. | **No.** Importing constants does not add manifests or render dispatch. |

Startup-file support also changes the relevance of the [README's manual archive/download workaround](../README.md): upstream now supports pre-boot extraction and automatic archive creation from a sibling folder on `labctl content push`, including watch mode. These are alternatives worth documenting once labx preserves the fields. Because labx emits into `dist/` and copies `static/` to `__static__/`, the sibling source folder must also be available beside the generated declaring file, or the archive must be prebuilt. Merely upgrading labx's module dependency does not update an independently installed `labctl` executable or stage those folders. Sources: [startup-file docs](https://labs.iximiuz.com/docs/custom-playgrounds/init-tasks#startup-files), [upstream push implementation](https://github.com/iximiuz/labctl/blob/v0.1.112/cmd/content/push.go), [labx content/static handling](../labx/content.go).

## Binding syntax and CLI installation

Native Labs double-brace bindings still use the documented raw-string escape in labx templates. Automatic syntax recognition is an optional future enhancement; variables already work with escaping or `:code-var`.

The Go module update does not update a separately installed `labctl` executable. Use the [official installation instructions](https://github.com/iximiuz/labctl#installation) to update that CLI.

[new-api]: https://github.com/iximiuz/labctl/blob/v0.1.112/api/playgrounds.go
[old-api]: https://github.com/iximiuz/labctl/blob/v0.1.72/api/playgrounds.go
