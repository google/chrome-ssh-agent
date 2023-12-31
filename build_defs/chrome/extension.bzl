load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

PACKAGES = ("chrome",)
PLATFORMS = ("linux64", "win64", "mac-x64")

def _chrome_download(package, platform, version):
    http_archive(
        name = "chrome_{package}_{platform}".format(
            package = package,
            platform = platform,
        ),
        build_file_content = """
filegroup(
    name = "pkg",
    data = glob(["{package}-{platform}/**"]),
    visibility = ["//visibility:public"],
)
        """.format(
            package = package,
            platform = platform,
        ),
        urls = [
            "https://edgedl.me.gvt1.com/edgedl/chrome/chrome-for-testing/{version}/{platform}/{package}-{platform}.zip".format(
                package = package,
                platform = platform,
                version = version,
                )],
    )


def _chrome_dependencies_impl(_ctx):
    for mod in _ctx.modules:
        for download in mod.tags.download:
            for package in PACKAGES:
                for platform in PLATFORMS:
                    _chrome_download(package, platform, download.version)

_download = tag_class(
    attrs = {
        "version": attr.string(),
    }
)

chrome = module_extension(
    implementation = _chrome_dependencies_impl,
    tag_classes = {
        "download": _download,
    }
)