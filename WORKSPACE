workspace(name = "chrome-ssh-agent")

load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "6b65cb7917b4d1709f9410ffe00ecf3e160edf674b78c54a894471320862184f",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.39.0/rules_go-v0.39.0.zip"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "727f3e4edd96ea20c29e8c2ca9e8d2af724d8c7778e7923a854b2c80952bc405",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.30.0/bazel-gazelle-v0.30.0.tar.gz"],
)

http_archive(
    name = "aspect_rules_js",
    sha256 = "aea8d12bdc4b40127e57fb3da5b61cbb17e969e7786471a71cbff0808c600bcb",
    strip_prefix = "rules_js-1.24.1",
    url = "https://github.com/aspect-build/rules_js/releases/download/v1.24.1/rules_js-v1.24.1.tar.gz",
)

http_archive(
    name = "aspect_rules_ts",
    sha256 = "8eb25d1fdafc0836f5778d33fb8eaac37c64176481d67872b54b0a05de5be5c0",
    strip_prefix = "rules_ts-1.3.3",
    url = "https://github.com/aspect-build/rules_ts/releases/download/v1.3.3/rules_ts-v1.3.3.tar.gz",
)

http_archive(
    name = "aspect_rules_esbuild",
    sha256 = "3e074ee7be579ceb4f0a664f6ae88fa68926e8eec65ffa067624c5d98c9552f6",
    strip_prefix = "rules_esbuild-0.13.5",
    url = "https://github.com/aspect-build/rules_esbuild/archive/refs/tags/v0.13.5.tar.gz",
)

http_archive(
    name = "bazel_skylib",
    sha256 = "b8a1527901774180afc798aeb28c4634bdccf19c4d98e7bdd1ce79d1fe9aaad7",
    urls = ["https://github.com/bazelbuild/bazel-skylib/releases/download/1.4.1/bazel-skylib-1.4.1.tar.gz"],
)

http_archive(
    name = "rules_pkg",
    sha256 = "335632735e625d408870ec3e361e192e99ef7462315caa887417f4d88c4c8fb8",
    urls = ["https://github.com/bazelbuild/rules_pkg/releases/download/0.9.0/rules_pkg-0.9.0.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "80d3a4ec17354cccc898bfe32118edd934f851b03029d63ef3fc7c8663a7415c",
    strip_prefix = "rules_proto-5.3.0-21.5",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/refs/tags/5.3.0-21.5.tar.gz",
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
    # Use semver-coerced to handle versions where patches are left off (e.g., 1.19).
    version = "1.20.3",  # renovate: datasource=golang-version depName=golang versioning=semver-coerced
)

# Gazelle dependency management support
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

# Pull in external dependencies:
# gazelle:repository_macro go_deps.bzl%go_dependencies
load("//:go_deps.bzl", "go_dependencies")

go_dependencies()

gazelle_dependencies()

# Javascript support.
load("@aspect_rules_js//js:repositories.bzl", "rules_js_dependencies")

rules_js_dependencies()

load("@rules_nodejs//nodejs:repositories.bzl", "DEFAULT_NODE_VERSION", "node_repositories", "nodejs_register_toolchains")

nodejs_register_toolchains(
    name = "nodejs",
    node_version = DEFAULT_NODE_VERSION,
)

load("@aspect_rules_js//npm:npm_import.bzl", "npm_translate_lock")

npm_translate_lock(
    name = "npm",
    pnpm_lock = "//:pnpm-lock.yaml",
    verify_node_modules_ignored = "//:.bazelignore",
)

load("@npm//:repositories.bzl", "npm_repositories")

npm_repositories()

# Typescript support.
load("@aspect_rules_ts//ts:repositories.bzl", "LATEST_VERSION", "rules_ts_dependencies")

rules_ts_dependencies(ts_version = LATEST_VERSION)

# esbuild support.

load("@aspect_rules_esbuild//esbuild:dependencies.bzl", "rules_esbuild_dependencies")

rules_esbuild_dependencies()

load("@aspect_rules_esbuild//esbuild:repositories.bzl", "LATEST_VERSION", "esbuild_register_toolchains")

esbuild_register_toolchains(
    name = "esbuild",
    esbuild_version = LATEST_VERSION,
)

# Skylib for helpful utilities in custom rules.

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

# Package building rules.
load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

rules_pkg_dependencies()
