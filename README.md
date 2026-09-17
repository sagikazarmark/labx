# labx

> I can solve it from the frontend.

**Opinionated tools for working with [iximiuz Labs](http://labs.iximiuz.com) content.**

> [!WARNING]
> This tool is still in development and hasn't been tested with all content types.

## Installation

```shell
go install github.com/sagikazarmark/labx@latest
```

> [!NOTE]
> Better installation methods will be provided once the tool becomes more stable.

## How does it work?

The tool processes playground or content manifests written in YAML, transforming and extending the core manifest format with additional features.

One key difference from the standard iximiuz Labs format: YAML frontmatter is handled as separate `.yaml` files.
This makes editing easier—especially when working side-by-side, rather than scrolling through a single file.

After processing the manifest, the tool compiles the final Markdown
and writes the output to a `dist/` directory relative to the source files.

## Features

### Play variables

Content manifests support Labs' [play variables](https://labs.iximiuz.com/tutorials/sample-tutorial#play-variables).
Declarations are passed through to Labs and resolved at runtime:

```yaml
vars:
  PORT: "8080"
  POD_IP: x(.tasks.init_lookup.stdout)
  URL: http://x(.vars.POD_IP):x(.vars.PORT)/
  SESSION_ID:
    shell: cat /proc/sys/kernel/random/uuid
    machine: dev-machine
```

Use `x(.vars.PORT)` in task scripts or environment values. References to tasks expanded across
machines/users must use their generated names (for example, `init_lookup_dev_machine_root`);
labx only rewrites `needs`, not expressions inside variable declarations or scripts.

In Markdown, escape Labs' double-brace bindings with a Go-template raw string so they survive
build-time rendering:

```markdown
Connect to {{ `{{ vars.URL || 'waiting...' }}` }}.
```

The generated Markdown contains `{{ vars.URL || 'waiting...' }}`. Alternatively, use the native
`:code-var{name=URL default='waiting...'}` directive, which needs no escaping. These runtime
variables are separate from labx's build-time `.Extra` data; `.Manifest.Vars` exposes declarations,
not their resolved values.

Content also preserves `helper: true` tasks, `playgrounds` and `skill-paths` embedding maps,
and `playground.registryAuth`. See the [labctl compatibility review](docs/labctl-review.md)
for details and upstream references.

### Content kinds

Both generation and render-only mode support `skill-path`, `blog-post`, `shell-gym`,
`roadmap`, and `vendor`, in addition to playgrounds, tutorials, challenges, courses, and trainings.
These kinds render `index.md`. Skill paths also render sibling `unit-*.md` files or Markdown
files from `units/` into the output root, preserving their front matter. Use only one location
for each unit filename.

Skill-path manifests support `difficulties: [easy, medium]`. Additional top-level content
front-matter fields are preserved in YAML and available to templates through `.Manifest.Metadata`.
labx passes these fields through; Labs validates their meaning.

Blog-post support consists of index rendering and metadata preservation. Blog-only fields are
not modeled without a confirmed public authoring schema.

### Shell Gyms

Shell Gym manifests have typed `shellgym` and `repetition` fields:

```yaml
kind: shell-gym
playground:
  name: ubuntu-26-04
  startupFiles:
    - path: /opt/shellgym/path
      source: __static__/path.tar.gz
      extract: true
      owner: root
shellgym:
  version: latest
  path: /opt/shellgym/path
  port: 63636
repetition: [1, 3, 7, 14, 30]
```

These settings are optional. labx leaves omitted settings out so Labs applies its defaults.
The archive extraction path must match `shellgym.path` (default `/opt/shellgym/path`). The
first playground machine hosts the gym, using its default login user as the observed shell user.
Templates can access `.Manifest.ShellGym` and `.Manifest.Repetition`.

Place the learning path beside `manifest.yaml` in `path/`, with `path.yaml`, numbered module
directories, `module.md`, and numbered unit directories containing `unit.md`. labx copies the
path files verbatim and builds their archive; Shellgym's own tasks, variables, and scripts
are handled by Shellgym at runtime. Keep the raw `path/` files uploadable—do not exclude
`path/` through `.labctlignore`—because Labs uses them for searchability.

See the [Shell Gym format reference](https://github.com/iximiuz/labs/blob/main/content-samples/claude/rules/shell-gym-format-docs.md)
and [example fixture](labx/testdata/shell-gym).

### Nested YAML metadata

Locally modeled playground specs, machines, users, startup files, content tasks, init tasks,
and Shell Gym configuration preserve additional YAML fields through their `Metadata` maps.
For example, `.Manifest.Playground.Metadata` exposes unmodeled playground fields to templates.
Known fields remain in their typed properties; build-only fields such as `fromFile`, `hostname`,
and `channels` are consumed by generation rather than copied back into the output.
This preservation applies to local YAML models, not every nested type imported from labctl's API.

### Startup files and archives

Playgrounds and content can declare `playground.startupFiles`, optionally selecting target
`machines`, as well as per-machine startup files. Both locations support `content`, labx's
`fromFile`, and the native `source` / `extract` fields:

```yaml
playground:
  startupFiles:
    - path: /etc/example.conf
      fromFile: config/example.conf
    - path: /opt/app
      source: __static__/app.tar.gz
      extract: true
      machines: [dev-machine]
```

With an `app/` directory beside `manifest.yaml`, labx stages it beside the generated declaring
file and builds `__static__/app.tar.gz`. `.tar` and `.tgz` are also supported. Files are stored
relative to that directory, with executable modes preserved and fixed timestamps for reproducible
archives. `.labctlignore` files inside source directories use labctl's glob rules (including
directory-only patterns); Git metadata and editor backup files are excluded from archives.
labctl can also rebuild supported archive declarations from the staged folders during push/watch.
To avoid uploading the raw source files separately, add `app/` to a `.labctlignore` beside
`manifest.yaml`; labx copies that file into the output when staging archive sources. This
upload exclusion does not exclude the folder's contents from its archive.
For Shell Gyms, leave `path/` uploadable as described above.

If no sibling source directory exists, put a prebuilt archive in `static/`; labx copies it to
`__static__/`. Remote `source` URLs pass through unchanged. When both a source directory and
a prebuilt archive exist, the directory is the source of truth. Runtime extraction is performed
by Labs before boot. Content overrides append playground-level startup files inherited from
the base playground, just as they do per-machine startup files.

### Port forwards

Standalone playground and content manifests preserve native port-forward declarations:

```yaml
playground:
  portForwards:
    - kind: local
      machine: dev-machine
      localPort: 18080
      remotePort: 8080
```

### Customize hostname ([#8](https://github.com/iximiuz/labs/issues/8))

You can set a custom `hostname` for any machine in a playground or content manifest by adding it to the machine spec:

```yaml
playground:
  machines:
    - name: ubuntu-01
      hostname: openbao
```

The specified hostname will be applied by generating a set of `startupFiles` that configure the machine accordingly.

### Legacy automatic downloads ([#24](https://github.com/iximiuz/labs/issues/24))

For new content, prefer the pre-boot startup-file archive support above. The existing init-task
download workflow is also available:

If a `dist/__static__/{KIND}.tar.gz` archive exists (where `{KIND}` is the content kind),
the tool automatically injects an init task into each machine to download and extract it to `/opt/{KIND}`.

You're responsible for creating the archive, giving you full control over how the content is structured.

### Run tasks on multiple machines and/or users ([#11](https://github.com/iximiuz/labs/issues/11))

Sometimes you need to run the same task on multiple machines, for multiple users (e.g., to configure authentication), or both.

The extended manifest format supports this by allowing multiple entries in the `machine` and `user` fields.
The tool will automatically expand these into a series of tasks for each combination.

Any dependencies listed under `needs` are also updated accordingly.

Omitting `machine` or `user` leaves that field omitted on the generated task so Labs can
choose its default. An omitted dimension still produces one task; an explicitly empty list
produces none. This applies to content `tasks` and playground `initTasks`.

Content manifests also support `playground.initTasks` and `playground.initConditions`.
Content tasks can depend on locally declared playground init tasks, including expanded tasks;
local init tasks can depend on named init tasks supplied by the base playground.

```yaml
  initTasks:
    init_openbao_auth:
      name: init_openbao_auth
      machine: node-01
      init: true
      user:
        - root
        - laborant
      run: echo iximiuz > ~/.bao-token
    init_install_proxy:
      name: init_install_proxy
      machine:
        - node-01
        - node-02
      init: true
      user: root
      needs:
        - init_files # This is also a multi-machine task, so it will be updated accordingly
      run: /opt/playground/proxy/install.sh
```

## Improved merging of machines ([#23](https://github.com/iximiuz/labs/issues/23))

Right now, if you define `machines` for a custom playground in any content, any machine configuration from the playground gets overwritten.

This tool improves merging by making sure:

- startup files are appended to the custom playground startup files
- users are copied from the playground when none are defined
- resource config is taken from the playground when undefined

> [!NOTE]
> This feature works by fetching the playground manifest from the server, so make sure to login with `labctl`.
