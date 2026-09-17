---
title: Change into a directory
vars:
  DIRNAME:
    pick: [alpha, bravo]
init:
  - name: create-directory
    run: mkdir -p /tmp/gym/$DIRNAME
tasks:
  change-directory:
    mode: edge
    check: wait_cwd "/tmp/gym/$DIRNAME"
    hint: echo "Use the shell's directory-changing command."
    solve: cd /tmp/gym/$DIRNAME
---

Change your current directory to `/tmp/gym/${DIRNAME}`.

::task{name="change-directory"}
#active
Waiting for your shell to enter the directory.
#completed
You reached the directory.
::

Shell strings such as `{{ not_a_labx_template }}` remain literal text in this learning path.
