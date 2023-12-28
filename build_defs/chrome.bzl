load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

_CHROME_VERSION = "122.0.6212.0"  # renovate: datasource=custom.chrome depName=linux64 versioning=loose

def chrome_data_dependencies():
    http_archive(
        name = "chrome",
        build_file_content =
            """
exports_files(["chrome-linux64/chrome"])
""",
        urls = ["https://edgedl.me.gvt1.com/edgedl/chrome/chrome-for-testing/{}/linux64/chrome-linux64.zip".format(_CHROME_VERSION)],
    )


def _chrome_dependencies_impl(_ctx):
    chrome_data_dependencies()

chrome_dependencies = module_extension(
    implementation = _chrome_dependencies_impl,
)