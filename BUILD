load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "TOOLS_NOGO", "nogo")
load("@rules_pkg//:pkg.bzl", "pkg_zip")

# gazelle:prefix github.com/google/chrome-ssh-agent
gazelle(name = "gazelle")

# Enable nogo for source code analysis
nogo(
    name = "chrome_ssh_agent_nogo",
    config = ":nogo-config.json",
    visibility = ["//visibility:public"],
    deps = TOOLS_NOGO,
)

exports_files([
    "go.mod",
    "go.sum",
    "package.json",
    "package-lock.json",
])

pkg_zip(
    name = "chrome-ssh-agent",
    srcs = [
        ":CONTRIBUTING.md",
        ":LICENSE",
        ":README.md",
":manifest.json",
        "//go/background:pkg",
        "//go/options:pkg",
        "//html:pkg",
        "//img:pkg",
    ],
    visibility = ["//visibility:public"],
)
