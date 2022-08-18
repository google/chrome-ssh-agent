workspace(name = "chrome-ssh-agent")

load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

http_archive(
    name = "io_bazel_rules_go",
    patch_args = ["-p1"],
    patches = [
        "//:patches/rules_go-wasm.patch",
    ],
    sha256 = "16e9fca53ed6bd4ff4ad76facc9b7b651a89db1689a2877d6fd7b82aa824e366",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.34.0/rules_go-v0.34.0.zip"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "501deb3d5695ab658e82f6f6f549ba681ea3ca2a5fb7911154b5aa45596183fa",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.26.0/bazel-gazelle-v0.26.0.tar.gz"],
)

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "ee3280a7f58aa5c1caa45cb9e08cbb8f4d74300848c508374daf37314d5390d6",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/5.5.1/rules_nodejs-5.5.1.tar.gz"],
)

http_archive(
    name = "bazel_skylib",
    sha256 = "f7be3474d42aae265405a592bb7da8e171919d74c16f082a5457840f06054728",
    urls = ["https://github.com/bazelbuild/bazel-skylib/releases/download/1.2.1/bazel-skylib-1.2.1.tar.gz"],
)

http_archive(
    name = "rules_pkg",
    sha256 = "8a298e832762eda1830597d64fe7db58178aa84cd5926d76d5b744d6558941c2",
    urls = ["https://github.com/bazelbuild/rules_pkg/releases/download/0.7.0/rules_pkg-0.7.0.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "e017528fd1c91c5a33f15493e3a398181a9e821a804eb7ff5acdd1d2d6c2b18d",
    strip_prefix = "rules_proto-4.0.0-3.20.0",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/refs/tags/4.0.0-3.20.0.tar.gz",
    ],
)


# Instructions, courtesy of rules_webtesting.
#
# To update Chromium, do the following:
# Step 1: Go to https://omahaproxy.appspot.com/
# Step 2: Look for branch_base_position of current stable releases
# Step 3: Go to https://commondatastorage.googleapis.com/chromium-browser-snapshots/index.html?prefix=Linux_x64/ etc to verify presence of that branch release for that platform.
#         If no results, delete the last digit to broaden your search til you find a result.
# Step 4: Verify both Chromium and ChromeDriver are released at that version.
# Step 5: Update the URL to the new release.
http_archive(
    name = "chromedriver",
    build_file_content =
        """
genrule(
    name = "chromedriver",
    srcs = [":chromedriver_linux64/chromedriver"],
    outs = ["chromedriver.bin"],
    cmd = "ln $(location //:chromedriver_linux64/chromedriver) $@",
    visibility = ["//visibility:public"],
)
""",
    sha256 = "30c27c17133bf3622f0716e1bc70017dc338a6920ea1b1f3eb15f407150b927c",
    # File within archive: chromedriver_linux64/chromedriver
    # 103.0.5060.134
    urls = ["https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/1002910/chromedriver_linux64.zip"],
)

http_archive(
    name = "chromium",
    build_file_content =
        """
genrule(
    name = "chromium",
    srcs = [":chrome-linux/chrome"],
    outs = ["chromium.bin"],
    cmd = "ln $(location //:chrome-linux/chrome) $@",
    visibility = ["//visibility:public"],
)
""",
    sha256 = "53899aaf90d9b9768dbc54beb869a314bdc8f4d04c2ef7bab2cb480581cfa197",
    # File within archive: chrome-linux/chrome
    # 103.0.5060.134
    urls = ["https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/1002910/chrome-linux.zip"],
)

# Proto support.  Required by github.com/mediabuyerbot/go-crx3, which is needed by
# Selenium browser-based testing.
#
# See https://github.com/bazelbuild/rules_go/issues/2902 for how to configure this
# such that we avoid needing to compile the proto compiler.  Avoid the standard
# instructions in https://github.com/bazelbuild/rules_go#protobuf-and-grpc.
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()
rules_proto_toolchains()

# Go build support.
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    nogo = "@//:chrome_ssh_agent_nogo",
    version = "1.18",
)

# Gazelle dependency management support
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

# Pull in external dependencies:
# gazelle:repository_macro go_deps.bzl%go_dependencies
load("//:go_deps.bzl", "go_dependencies")

go_dependencies()

gazelle_dependencies()

# NodeJS support. Required for testing and publishing.
load("@build_bazel_rules_nodejs//:repositories.bzl", "build_bazel_rules_nodejs_dependencies")

build_bazel_rules_nodejs_dependencies()

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories", "npm_install")

node_repositories()

npm_install(
    name = "npm",
    package_json = "//:package.json",
    package_lock_json = "//:package-lock.json",
)

# Skylib for helpful utilities in custom rules.

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

# Package building rules.
load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

rules_pkg_dependencies()
