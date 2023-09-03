load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def chromium_data_dependencies():
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


def _chromium_dependencies_impl(_ctx):
    chromium_data_dependencies()

chromium_dependencies = module_extension(
    implementation = _chromium_dependencies_impl,
)