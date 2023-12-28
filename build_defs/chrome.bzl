load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def _chrome_download(download):
    http_archive(
        name = "{package}_{platform}".format(
            package = download.package,
            platform = download.platform,
        ),
        build_file_content = """exports_files(["{package}-{platform}/chrome"])""".format(
            package = download.package,
            platform = download.platform,
        ),
        urls = [
            "https://edgedl.me.gvt1.com/edgedl/chrome/chrome-for-testing/{version}/{platform}/{package}-{platform}.zip".format(
                package = download.package,
                platform = download.platform,
                version = download.version,
                )],
    )


def _chrome_dependencies_impl(_ctx):
    for mod in _ctx.modules:
        for download in mod.tags.download:
            _chrome_download(download)

_download = tag_class(
    attrs = {
        "package": attr.string(),
        "platform": attr.string(),
        "version": attr.string(),
    }
)

chrome = module_extension(
    implementation = _chrome_dependencies_impl,
    tag_classes = {
        "download": _download,
    }
)