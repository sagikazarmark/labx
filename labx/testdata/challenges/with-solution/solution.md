::remark-box
💡 You can review the changes made by the commands below by checking the `~/my-module/dagger.json` file.
::

## Preparation

Initialize a new Dagger module at `~/my-module`:

```shell
{{ readFileBlock "solution/solve.sh" "init" }}
```

## Installing modules

Install the `go` module:

```shell
{{ readFileBlock "solution/solve.sh" "install_go" }}
```

Install any module (other than `go`) as your "favorite" module:

```shell
{{ readFileBlock "solution/solve.sh" "install_favorite" }}
```

::remark-box
💡 Note that you can install modules either with a specific version or without one: `github.com/username/repo[/subdir][@version]`
::

## Updating modules

Install version `0.13.0` of the [helm](https://daggerverse.dev/mod/github.com/sagikazarmark/daggerverse/helm) module:

```shell
{{ readFileBlock "solution/solve.sh" "install_helm" }}
```

Check `dagger.json` to review the newly added dependency:

```shell
jq -r '.dependencies[] | select(.name == "helm")' ~/my-module/dagger.json
```

Try updating the `helm` module:

```shell
{{ readFileBlock "solution/solve.sh" "update_helm" }}
```

Notice that the version hasn't changed:

```shell
jq -r '.dependencies[] | select(.name == "helm")' ~/my-module/dagger.json
```

That's because the module is pinned to a specific version (`0.13.0` in this case).

You can remove it using this script (or in the :tab-locator-inline{text='IDE' name='IDE'}):

```shell
{{ readFileBlock "solution/solve.sh" "unpin_helm" }}
```

Try updating the `helm` module again:

```shell
{{ readFileBlock "solution/solve.sh" "update_helm" }}
```

The `pin` field should reflect the updated version of the `helm` module:

```shell
jq -r '.dependencies[] | select(.name == "helm")' ~/my-module/dagger.json
```

## Uninstalling modules

Install the [helm-docs](https://daggerverse.dev/mod/github.com/sagikazarmark/daggerverse/helm-docs) module:

```shell
{{ readFileBlock "solution/solve.sh" "install_helm_docs" }}
```

Now uninstall it:

```shell
{{ readFileBlock "solution/solve.sh" "uninstall_helm_docs" }}
```
