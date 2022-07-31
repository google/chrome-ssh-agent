package test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	slog "github.com/tebeka/selenium/log"
)

func mustRunfile(path string) string {
	path, err := bazel.Runfile(path)
	if err != nil {
		panic(fmt.Errorf("failed to find runfile %s: %v", path, err))
	}
	return path
}

var (
	chromeDriverPath = mustRunfile("chromedriver.bin")
	chromePath       = mustRunfile("chromium.bin")
	extensionPath    = mustRunfile("chrome-ssh-agent.zip")
)

func getElementText(wd selenium.WebDriver, id string) (string, error) {
	el, err := wd.FindElement(selenium.ByID, id)
	if err != nil {
		return "", fmt.Errorf("Failed to find element with ID %s: %v", id, err)
	}

	txt, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("Failed to get text for element with ID %s: %v", id, err)
	}

	return txt, nil
}

func elementExists(id string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(selenium.ByID, id)
		return err == nil, nil
	}
}

func currentURLIs(url string) selenium.Condition {
	return func(wd selenium.WebDriver) (bool, error) {
		u, err := wd.CurrentURL()
		if err != nil {
			return false, err
		}
		return url == u, nil
	}
}

var (
	logLevels = slog.Capabilities{
		slog.Browser:     slog.All,
		slog.Performance: slog.Info,
		slog.Driver:      slog.Info,
	}
)

func dumpSeleniumLogs(t *testing.T, wd selenium.WebDriver) {
	t.Log("Dumping Selenium Logs")
	for typ, _ := range logLevels {
		msgs, err := wd.Log(typ)
		if err != nil {
			t.Errorf("Failed to fetch logs of type %s: %v", typ, err)
		}
		for _, msg := range msgs {
			t.Logf("SeleniumLog[%s]: %s [%s] %s", typ, msg.Timestamp, msg.Level, msg.Message)
		}
	}
}

func unusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func dumpLog(t *testing.T, name string, r io.Reader) {
	t.Logf("Dumping log: %s", name)
	if _, err := io.Copy(os.Stderr, r); err != nil {
		t.Errorf("Failed to dump log %s: %v", name, err)
	}
}

func TestWebApp(t *testing.T) {
	port, err := unusedPort()
	if err != nil {
		t.Fatalf("failed to identify unused port: %v", err)
	}

	var selOut bytes.Buffer
	opts := []selenium.ServiceOption{
		selenium.StartFrameBuffer(),
		selenium.Output(&selOut),
	}
	service, err := selenium.NewChromeDriverService(chromeDriverPath, port, opts...)
	if err != nil {
		defer dumpLog(t, "SeleniumOutput", &selOut)  // Selenium failed to initialize; show debug info.
		t.Fatalf("failed to start Selenium service: %v", err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{}
	caps.AddLogging(logLevels)

	t.Log("Preparing extension")
	extPath, extCleanup, err := unzipExtension(extensionPath)
	if err != nil {
		t.Fatalf("Failed to unzip extension: %v", err)
	}
	defer extCleanup()

	t.Log("Configuring extension in Chrome")
	chromeCaps := chrome.Capabilities{
		Path: chromePath,
		Args: []string{
			"--no-sandbox",
		},
	}
	if err = chromeCaps.AddUnpackedExtension(extPath); err != nil {
		t.Fatalf("failed to add extension: %v", err)
	}
	caps.AddChrome(chromeCaps)

	t.Log("Starting WebDriver")
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		defer dumpLog(t, "SeleniumOutput", &selOut)  // Selenium failed to initialize; show debug info.
		t.Fatalf("Failed to start webdriver: %v", err)
	}
	defer wd.Quit()
	defer dumpSeleniumLogs(t, wd)

	t.Log("Navigating to test page")
	path := makeExtensionUrl("html/options.html", "test")
	if err = wd.Get(path.String()); err != nil {
		t.Fatalf("Failed to navigate to %s: %v", path, err)
	}

	t.Log("Waiting for navigation")
	if err = wd.WaitWithTimeout(currentURLIs(path.String()), 10*time.Second); err != nil {
		t.Fatalf("Failed to complete navigation to page: %v", err)
	}

	src, err := wd.PageSource()
	if err != nil {
		t.Fatalf("Failed to retrieve page source: %v", err)
	}
	t.Logf("Page source:\n%s", src)

	t.Log("Waiting for results")
	if err = wd.WaitWithTimeout(elementExists("failureCount"), 10*time.Second); err != nil {
		t.Fatalf("failed to wait for failure count: %v", err)
	}
	if err = wd.WaitWithTimeout(elementExists("failures"), 10*time.Second); err != nil {
		t.Fatalf("failed to wait for failures: %v", err)
	}

	t.Log("Extracting test results")
	countTxt, err := getElementText(wd, "failureCount")
	if err != nil {
		t.Fatalf("Failed to find failure count: %v", err)
	}

	count, err := strconv.Atoi(countTxt)
	if err != nil {
		t.Fatalf("Failed to parse failure count '%s' as integer: %v", countTxt, err)
	}

	failures, err := getElementText(wd, "failures")
	if err != nil {
		t.Fatalf("Failed to find failure details: %v", err)
	}

	if count != 0 {
		t.Errorf("Reported Failures:\n%s", failures)
	}
}
