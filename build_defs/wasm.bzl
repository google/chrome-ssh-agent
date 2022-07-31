load("@io_bazel_rules_go//go/private:context.bzl", "go_context")
load("@io_bazel_rules_go//go/private:providers.bzl", "GoLibrary", "GoPath", "GoArchive")
load("@io_bazel_rules_go//go:def.bzl", "go_path")
load("@bazel_skylib//lib:paths.bzl", "paths")

def _go_wasm_test_impl(ctx):
    go = go_context(ctx)
    node_info = ctx.toolchains["@rules_nodejs//nodejs:toolchain_type"].nodeinfo
    node_path = node_info.target_tool_path
    node_inputs = node_info.tool_files

    runner = ctx.actions.declare_file(ctx.label.name + '_runner.sh')
    ctx.actions.write(
        runner,
        '\n'.join([
            '#!/bin/bash -eu',
            # Setup Go environment
	    'export GOROOT="${{PWD}}/{0}"'.format(go.env['GOROOT']),
	    'export GOPATH="${{PWD}}/{0}"'.format(ctx.attr.gopath[GoPath].gopath),
	    'export GOCACHE=${PWD}/.gocache',
	    'export GO111MODULE=off',  # We populated GOPATH; no need for modules
            # Setup for WASM
            'export GOOS=js',
            'export GOARCH=wasm',
            # Ensure 'node' binary is on the PATH.
            'export PATH="${{PWD}}/{0}:${{PATH}}"'.format(paths.dirname(node_path)),
            # TODO: Replace with proper way to discover directory
            'export NODE_PATH="${PWD}/external/npm/node_modules"',
	    'exec ${{PWD}}/{0} test -exec="${{PWD}}/{1}" {2}'.format(
                go.go.short_path,
                ctx.executable.run_wasm.short_path,
                ctx.attr.testlib[GoLibrary].importpath,
            ),
        ]),
        is_executable = True,
    )

    runfiles = ctx.runfiles(files=(
        go.sdk.headers
        + go.sdk.srcs
        + node_inputs
        + ctx.files.node_deps
        + [ctx.executable.run_wasm]
	+ [go.go] + go.sdk.tools
    ))
    runfiles = runfiles.merge(ctx.attr.gopath[DefaultInfo].default_runfiles)
    runfiles = runfiles.merge(ctx.attr.testlib[GoArchive].runfiles)

    return [DefaultInfo(
        executable = runner,
        runfiles = runfiles,
    )]


_go_wasm_test = rule(
    implementation = _go_wasm_test_impl,
    test = True,
    attrs = {
        "testlib": attr.label(
            mandatory = True,
            providers = [GoLibrary, GoArchive],
        ),
	"gopath": attr.label(
            mandatory = True,
            providers = [GoPath],
        ),
        "run_wasm": attr.label(
            executable = True,
            cfg = "exec",
            allow_files = True,
            default = Label("@go_sdk//:misc/wasm/go_js_wasm_exec"),
        ),
        "node_deps": attr.label_list(), 
        "_go_context_data": attr.label(
            default = "@io_bazel_rules_go//:go_context_data",
        ),
    },
    toolchains = [
        "@io_bazel_rules_go//go:toolchain",
	"@rules_nodejs//nodejs:toolchain_type",
    ],
)

def go_wasm_test(name, testlib, **kwargs):
    path_target = '{0}_gopath'.format(name)
    go_path(
        name = path_target,
        deps = [testlib],
	testonly = True,
    )
    _go_wasm_test(
        name = name,
	gopath = ':{0}'.format(path_target),
        testlib = testlib,
	**kwargs,
    )
