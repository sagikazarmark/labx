## Preparation

To complete this challenge, ensure you have a Dagger module initialized at `~/my-module`.

::remark-box
💡 You don't need to specify an SDK for this challenge.
::

::simple-task
---
:tasks: tasks
:name: verify_module_initialized
---
#active
Waiting for a module to be initialized at `~/my-module`...

#completed
Found module at `~/my-module`.
::

## Installing modules

Install the [go](https://daggerverse.dev/mod/github.com/sagikazarmark/daggerverse/go) module from the [Daggerverse](https://daggerverse.dev).

::simple-task
---
:tasks: tasks
:name: verify_go_module_installed
---
#active
Waiting for the `go` module to be installed...

#completed
`go` module installed.
::

::hint-box
---
:summary: Hint 1
---

Module path: `github.com/sagikazarmark/daggerverse/go`
::

::hint-box
---
:summary: Hint 2
---

Not sure which command to use? Try running `dagger --help` to see all available options.
::

::hint-box
---
:summary: Hint 3
---

`dagger install` is the command you need.
::

Navigate to the :tab-locator-inline{text='Daggerverse' name='Daggerverse'},
choose a module you like, and install it using the name `my-favorite`.

::simple-task
---
:tasks: tasks
:name: verify_favorite_module_installed
---
#active
Waiting for favorite module to be installed...

#completed
Favorite module installed.
::

::hint-box
---
:summary: Hint 4
---

You can specify a custom name using the `--name` flag.
::

## Updating modules

Install an earlier release of the [helm](https://daggerverse.dev/mod/github.com/sagikazarmark/daggerverse/helm) module (for example, version `0.13.0`).

::simple-task
---
:tasks: tasks
:name: verify_helm_module_installed
---
#active
Waiting for the `helm` module to be installed...

#completed
`helm` module installed.
::

::hint-box
---
:summary: Hint 5
---

Module path: `github.com/sagikazarmark/daggerverse/helm@v0.13.0`
::

Check `dagger.json` to review the newly added dependency:

```shell
jq -r '.dependencies[] | select(.name == "helm")' ~/my-module/dagger.json
```

Update the `helm` module to the latest version.

::hint-box
---
:summary: Hint 6
---

Not sure which command to use? `dagger --help` is still there.
::

::hint-box
---
:summary: Hint 7
---

`dagger update` seems useful.
::

Did anything change? Let's take another look at the `dagger.json` file:

```shell
jq -r '.dependencies[] | select(.name == "helm")' ~/my-module/dagger.json
```

What could be the problem?

::hint-box
---
:summary: Hint 8
---

There is a version in the `source` field.
::

::hint-box
---
:summary: Hint 9
---

Have you tried removing the `@helm/v0.13.0` bit from the `source` field?

You can use the :tab-locator-inline{text='IDE' name='IDE'} to edit the file or run this script:

```shell
jq '.dependencies |= map(if .name == "helm" and has("source") then .source |= sub("@.*$"; "") else . end)' ~/my-module/dagger.json | sponge ~/my-module/dagger.json
```
::

Try updating the module again.

::simple-task
---
:tasks: tasks
:name: verify_helm_module_updated
---
#active
Waiting for the `helm` module to be updated...

#completed
`helm` module updated.
::

## Uninstalling modules

Install the [helm-docs](https://daggerverse.dev/mod/github.com/sagikazarmark/daggerverse/helm-docs) module.

::simple-task
---
:tasks: tasks
:name: verify_helm_docs_module_installed
---
#active
Waiting for the `helm-docs` module to be installed...

#completed
`helm-docs` module installed.
::

::hint-box
---
:summary: Hint 10
---

Module path: `github.com/sagikazarmark/daggerverse/helm-docs`
::

Now uninstall the `helm-docs` module.

::simple-task
---
:tasks: tasks
:name: verify_helm_docs_module_uninstalled
---
#active
Waiting for the `helm-docs` module to be uninstalled...

#completed
`helm-docs` module uninstalled.
::

::hint-box
---
:summary: Hint 11
---

If you used `dagger install` to add a module, what command do you think you'd use to remove one?
::

::hint-box
---
:summary: Hint 12
---

Maybe check `dagger --help` again?
::

::hint-box
---
:summary: Hint 13
---

`dagger uninstall` seems like a logical choice.
::

## References

- Dagger documentation
  - [Remote Repositories](https://docs.dagger.io/api/remote-repositories)
  - [Remote Modules](https://docs.dagger.io/api/remote-modules)
- [Daggerverse](https://daggerverse.dev)
