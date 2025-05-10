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

### Customize hostname ([#8](https://github.com/iximiuz/labs/issues/8))

You can set a custom `hostname` for any machine in a playground or content manifest by adding it to the machine spec:

```yaml
playground:
  machines:
    - name: ubuntu-01
      hostname: openbao
```

The specified hostname will be applied by generating a set of `startupFiles` that configure the machine accordingly.

### Automatically download files ([#24](https://github.com/iximiuz/labs/issues/24))

Sometimes, you need to download files to the machine. This tool automates that step.

If a `dist/__static__/{KIND}.tar.gz` archive exists (where `{KIND}` is the content kind),
the tool automatically injects an init task into each machine to download and extract it to `/opt/{KIND}`.

You're responsible for creating the archive, giving you full control over how the content is structured.

### Run tasks on multiple machines and/or users ([#11](https://github.com/iximiuz/labs/issues/11))

Sometimes you need to run the same task on multiple machines, for multiple users (e.g., to configure authentication), or both.

The extended manifest format supports this by allowing multiple entries in the `machine` and `user` fields.
The tool will automatically expand these into a series of tasks for each combination.

Any dependencies listed under `needs` are also updated accordingly.

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
