package test

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bazelbuild/rules_webtesting/go/webtest"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	slog "github.com/tebeka/selenium/log"
)

const (
	wslCapabilitiesKey     = "google:wslConfig"
	wslCapabilitiesArgsKey = "args"
)

type wslCapabilities map[string]interface{}

func addChromeDriverArgs(caps selenium.Capabilities, args ...string) error {
	if _, ok := caps[wslCapabilitiesKey]; !ok {
		caps[wslCapabilitiesKey] = wslCapabilities{}
	}
	m, ok := caps[wslCapabilitiesKey].(wslCapabilities)
	if !ok {
		return fmt.Errorf("incorrect type for %s; got %T", wslCapabilitiesKey, caps[wslCapabilitiesKey])
	}

	if _, ok = m[wslCapabilitiesArgsKey]; !ok {
		m[wslCapabilitiesArgsKey] = []string{}
	}
	prevArgs, ok := m[wslCapabilitiesArgsKey].([]string)
	if !ok {
		return fmt.Errorf("incorrect type for %s; got %T", wslCapabilitiesArgsKey, m[wslCapabilitiesArgsKey])
	}
	m[wslCapabilitiesArgsKey] = append(prevArgs, args...)
	return nil
}

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

func dumpLogFile(t *testing.T, typ string, r io.Reader, maxTokenLength int) {
	t.Logf("Dumping log %s", typ)
	scanner := bufio.NewScanner(r)
	scanner.Buffer([]byte{}, maxTokenLength)
	for scanner.Scan() {
		t.Logf("%s: %s", typ, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Errorf("Failed to dump %s logs from file: %v", typ, err)
	}
}

func TestWebApp(t *testing.T) {
	caps := selenium.Capabilities{}
	caps.AddLogging(logLevels)

	t.Log("Configuring ChromeDriver log")
	driverLog, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("failed to create Chromedriver log temp file: %v", err)
	}
	defer os.Remove(driverLog.Name())
	if err = addChromeDriverArgs(caps, fmt.Sprintf("--log-path=%s", driverLog.Name())); err != nil {
		t.Fatalf("failed to add chrome driver args: %v", err)
	}

	t.Log("Preparing extension")
	extensionPath, extensionCleanup, err := unzipExtension()
	if err != nil {
		t.Fatalf("Failed to unzip extension: %v", err)
	}
	defer extensionCleanup()

	t.Log("Configuring extension in Chrome")
	chromeCaps := chrome.Capabilities{}
	if err = chromeCaps.AddUnpackedExtension(extensionPath); err != nil {
		t.Fatalf("failed to add extension: %v", err)
	}
	caps.AddChrome(chromeCaps)

	// Extension data is present in the ChromeDriver log. We need to bound
	// the size of a line present in the log.
	var extensionDataSize int
	for _, e := range chromeCaps.Extensions {
		extensionDataSize += len(e)
	}

	t.Log("Starting WebDriver")
	wd, err := webtest.NewWebDriverSession(caps)
	if err != nil {
		dumpLogFile(t, "ChromeDriverLog", driverLog, int(float64(extensionDataSize)*1.1))
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
	if err = wd.WaitWithTimeoutAndInterval(currentURLIs(path.String()), 10*time.Second, 1*time.Second); err != nil {
		t.Fatalf("Failed to complete navigation to page: %v", err)
	}

	src, err := wd.PageSource()
	if err != nil {
		t.Fatalf("Failed to retrieve page source: %v", err)
	}
	t.Logf("Page source:\n%s", src)

	t.Log("Waiting for results")
	if err = wd.WaitWithTimeoutAndInterval(elementExists("failureCount"), 10*time.Second, 1*time.Second); err != nil {
		t.Fatalf("failed to wait for failure count: %v", err)
	}
	if err = wd.WaitWithTimeoutAndInterval(elementExists("failures"), 10*time.Second, 1*time.Second); err != nil {
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
