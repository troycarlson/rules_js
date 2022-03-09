# Python Gazelle plugin

This directory contains a plugin for
[Gazelle](https://github.com/bazelbuild/bazel-gazelle)
that generates BUILD file content for Python code.

## Installation

First, you'll need to add Gazelle to your `WORKSPACE` file.
Follow the instructions at https://github.com/bazelbuild/bazel-gazelle#running-gazelle-with-bazel

Next, we need to fetch the third-party Go libraries that the python extension
depends on.

Add this to your `WORKSPACE`:

```starlark
# To compile the rules_python gazelle extension from source,
# we must fetch some third-party go dependencies that it uses.
load("@rules_python//gazelle:deps.bzl", _py_gazelle_deps = "gazelle_deps")

_py_gazelle_deps()
```

Next, we'll fetch metadata about your Python dependencies, so that gazelle can
determine which package a given import statement comes from. This is provided
by the `modules_mapping` rule. We'll make a target for consuming this
`modules_mapping`, and writing it as a manifest file for Gazelle to read.
This is checked into the repo for speed, as it takes some time to calculate
in a large monorepo.

Create a file `gazelle_python.yaml` next to your `requirements.txt`
file. (You can just use `touch` at this point, it just needs to exist.)

Then put this in your `BUILD.bazel` file next to the `requirements.txt`:

```starlark
load("@pip//:requirements.bzl", "all_whl_requirements")

# This rule fetches the metadata for python packages we depend on. That data is
# required for the gazelle_python_manifest rule to update our manifest file.
modules_mapping(
    name = "modules_map",
    wheels = all_whl_requirements,
)

# Gazelle python extension needs a manifest file mapping from
# an import to the installed package that provides it.
# This macro produces two targets:
# - //:gazelle_python_manifest.update can be used with `bazel run`
#   to recalculate the manifest
# - //:gazelle_python_manifest.test is a test target ensuring that
#   the manifest doesn't need to be updated
gazelle_python_manifest(
    name = "gazelle_python_manifest",
    modules_mapping = ":modules_map",
    # This is what we called our `pip_install` rule, where third-party
    # python libraries are loaded in BUILD files.
    pip_repository_name = "pip",
    # When using pip_parse instead of pip_install, set the following.
    # pip_repository_incremental = True,
    # This should point to wherever we declare our python dependencies
    # (the same as what we passed to the modules_mapping rule in WORKSPACE)
    requirements = "//:requirements_lock.txt",
)
```

That's it, now you can finally run `bazel run //:gazelle` anytime
you edit Python code, and it should update your `BUILD` files correctly.

A fully-working example is in [`examples/build_file_generation`](examples/build_file_generation).

## Usage

Gazelle is non-destructive.
It will try to leave your edits to BUILD files alone, only making updates to `py_*` targets.
However it will remove dependencies that appear to be unused, so it's a
good idea to check in your work before running Gazelle so you can easily
revert any changes it made.

The rules_python extension assumes some conventions about your Python code.
These are noted below, and might require changes to your existing code.

Note that the `gazelle` program has multiple commands. At present, only the `update` command (the default) does anything for Python code.

### Directives

You can configure the extension using directives, just like for other
languages. These are just comments in the `BUILD.bazel` file which
govern behavior of the extension when processing files under that
folder.

See https://github.com/bazelbuild/bazel-gazelle#directives
for some general directives that may be useful.
In particular, the `resolve` directive is language-specific
and can be used with Python.
Examples of these directives in use can be found in the
/gazelle/testdata folder in the aspect-build/rules_js repo.

TODO TypeScript-specific directives are as follows:

| **Directive**                        | **Default value** |
|--------------------------------------|-------------------|
| `# gazelle:typescript_*`             |       ?????       |
| TODO: list directives. | |

### Libraries

TypeScript source files are those ending in `.ts`.

TODO: differenciate source vs spec files?

First, we look for the nearest ancestor BUILD file starting from the folder
containing the Python source file.

If there is no `ts_project` in this BUILD file, one is created, using the
package name as the target's name. This makes it the default target in the
package.

Next, all source files are collected into the `srcs` of the `ts_project`.

Finally, the `import` statements in the source files are parsed, and
dependencies are added to the `deps` attribute.

TODO: require statements?

### TODO: Tests - *.spec.ts, ?

## Developing on the extension

Gazelle extensions are written in Go.

The Go dependencies are managed by the go.mod file.
After changing that file, run `go mod tidy` to get a `go.sum` file,
then run `bazel run //:update_go_deps` to convert that to the `gazelle/deps.bzl` file.
The latter is loaded in our `/WORKSPACE` to define the external repos
that we can load Go dependencies from.

Then after editing Go code, run `bazel run //:gazelle` to generate/update
go_* rules in the BUILD.bazel files in our repo.
