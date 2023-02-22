load("@io_bazel_rules_go//go/private:providers.bzl", "GoLibrary", "GoPath", "GoArchive")
load("@io_bazel_rules_go//go:def.bzl", "go_test", "go_library", "go_binary")
load("@bazel_skylib//lib:paths.bzl", "paths")


def _go_wasm_test_impl(ctx):
    node_info = ctx.toolchains["@rules_nodejs//nodejs:toolchain_type"].nodeinfo
    node_path = node_info.target_tool_path
    node_inputs = node_info.tool_files
    test_executable = ctx.attr.test_target[DefaultInfo].files_to_run.executable
    test_runfiles = ctx.attr.test_target[DefaultInfo].default_runfiles

    runner = ctx.actions.declare_file(ctx.label.name + '_runner.sh')
    node_module_paths = ctx.host_configuration.host_path_separator.join(_node_paths(
        depset(ctx.files.node_deps)
    ))
    ctx.actions.write(
        runner,
        '\n'.join([
            '#!/bin/bash -eu',
            # Ensure 'node' binary is on the PATH.
            'export PATH="${{PWD}}/{0}:${{PATH}}"'.format(paths.dirname(node_path)),
            'export NODE_PATH="${PWD}/node_modules"',
	    # Wrapping executes a subprocess and uses pipe() for communication;
	    # pipe() is unsupported under node.js and WASM.
	    'export GO_TEST_WRAP=0',
	    'exec ${{PWD}}/{0} "${{PWD}}/{1}"'.format(
                ctx.executable.run_wasm.short_path,
		test_executable.short_path,
            ),
        ]),
        is_executable = True,
    )

    runfiles = ctx.runfiles(files=(
	[runner, test_executable, ctx.executable.run_wasm]
        + node_inputs
    ))
    for nd in ctx.attr.node_deps:
        runfiles = runfiles.merge(nd[DefaultInfo].default_runfiles)
    runfiles = runfiles.merge(test_runfiles)

    return [DefaultInfo(
        executable = runner,
        runfiles = runfiles,
    )]


_go_wasm_test = rule(
    implementation = _go_wasm_test_impl,
    test = True,
    attrs = {
	"test_target": attr.label(
            mandatory = True,
            providers = [DefaultInfo],
        ),
        "run_wasm": attr.label(
            executable = True,
            cfg = "exec",
            allow_files = True,
            default = Label("@go_sdk//:misc/wasm/go_js_wasm_exec"),
        ),
        "node_deps": attr.label_list(), 
    },
    toolchains = [
	"@rules_nodejs//nodejs:toolchain_type",
    ],
)

def go_wasm_test(name, srcs, embed, deps, node_deps = [], **kwargs):
    # Define a target that builds the test binary.
    test_target = '_{0}_internal'.format(name)
    go_test(
        name = test_target,
	srcs = srcs,
	deps = deps,
	embed = embed,
	goos = "js",
	goarch = "wasm",
        # We don't want this target executed automatically with invocations
        # such as 'blaze build path/to/....', since it would not be executed
        # with the correct wrapper. To avoid this, add the 'manual' tag.
        tags = kwargs.get("tags", []) + ["manual"],
	**kwargs,
    )

    # Run the test binary via a wrapper.
    _go_wasm_test(
        name = name,
	test_target = ":{0}".format(test_target),
	node_deps = node_deps,
    )


def go_wasm_binary(name, **kwargs):
    if "out" not in kwargs:
        kwargs["out"] = "{0}.wasm".format(name)

    go_binary(
        name = name,
	goos = "js",
	goarch = "wasm",
        **kwargs,
    )
