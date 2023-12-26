load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def chromium_data_dependencies():
    # Instructions, courtesy of rules_webtesting.
    #
    # To update Chromium, do the following:
    # Step 1: Go to Go to https://chromiumdash.appspot.com/branches
    # Step 2: Look for branch_base_position of current stable releases
    # Step 3: Go to https://commondatastorage.googleapis.com/chromium-browser-snapshots/index.html?prefix=Linux_x64/ etc to verify presence of that branch release for that platform.
    #         If no results, delete the last digit to broaden your search til you find a result.
    # Step 4: Update the URL to the new release.
    http_archive(
        name = "chromium",
        build_file_content =
            """
exports_files(["chrome-linux/chrome"])
""",
        sha256 = "20eb493492f5384be8b16a04aaccb7fd9e712fb994c060a17791cd85048fa205",
        # Release Milestone: 118
        urls = ["https://storage.googleapis.com/chromium-browser-snapshots/Linux_x64/1192597/chrome-linux.zip"],
    )


def _chromium_dependencies_impl(_ctx):
    chromium_data_dependencies()

chromium_dependencies = module_extension(
    implementation = _chromium_dependencies_impl,
)