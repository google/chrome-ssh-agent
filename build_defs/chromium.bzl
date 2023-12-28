load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

_CHROME_VERSION = "122.0.6210.0"  # renovate: datasource=custom.chrome-linux64 depName=chrome-linux64 versioning=semver-coerced

def chromium_data_dependencies():
    http_archive(
        name = "chromium",
        build_file_content =
            """
exports_files(["chrome-linux64/chrome"])
""",
        urls = ["https://edgedl.me.gvt1.com/edgedl/chrome/chrome-for-testing/{}/linux64/chrome-linux64.zip".format(_CHROME_VERSION)],
    )


def _chromium_dependencies_impl(_ctx):
    chromium_data_dependencies()

chromium_dependencies = module_extension(
    implementation = _chromium_dependencies_impl,
)