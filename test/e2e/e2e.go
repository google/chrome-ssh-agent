package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/chrome-ssh-agent/go/testutil"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	slog "github.com/tebeka/selenium/log"
)

var (
	chromeDriverPath = testutil.MustRunfile("_main~chromium_dependencies~chromedriver/chromedriver_linux64/chromedriver")
	chromePath       = testutil.MustRunfile("_main~chromium_dependencies~chromium/chrome-linux/chrome")
)

func getElementText(wd selenium.WebDriver, id string) (string, error) {
	el, err := wd.FindElement(selenium.ByID, id)
	if err != nil {
		return "", fmt.Errorf("Failed to find element with ID %s: %w", id, err)
	}

	txt, err := el.Text()
	if err != nil {
		return "", fmt.Errorf("Failed to get text for element with ID %s: %w", id, err)
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

var logLevels = slog.Capabilities{
	slog.Browser:     slog.All,
	slog.Performance: slog.Info,
	slog.Driver:      slog.Info,
}

func dumpSeleniumLogs(t *testing.T, wd selenium.WebDriver) {
	t.Log("Dumping Selenium Logs")
	for typ := range logLevels {
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
	t.Parallel()

	testcases := []struct {
		name          string
		extensionPath string
		extensionID   string
	}{
		{
			name:          "Prod Release",
			extensionPath: testutil.MustRunfile("_main/chrome-ssh-agent.zip"),
			extensionID:   "eechpbnaifiimgajnomdipfaamobdfha",
		},
		{
			name:          "Beta Release",
			extensionPath: testutil.MustRunfile("_main/chrome-ssh-agent-beta.zip"),
			extensionID:   "onabphcdiffmanfdhkihllckikaljmhh",
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			port, err := unusedPort()
			if err != nil {
				t.Fatalf("failed to identify unused port: %v", err)
			}

			var selOut bytes.Buffer
			opts := []selenium.ServiceOption{
				selenium.Output(&selOut),
			}
			service, err := selenium.NewChromeDriverService(chromeDriverPath, port, opts...)
			if err != nil {
				defer dumpLog(t, "SeleniumOutput", &selOut) // Selenium failed to initialize; show debug info.
				t.Fatalf("failed to start Selenium service: %v", err)
			}
			defer func() {
				if serr := service.Stop(); serr != nil {
					t.Errorf("failed to stop Selenium service: %v", serr)
				}
			}()

			caps := selenium.Capabilities{}
			caps.AddLogging(logLevels)

			t.Log("Preparing extension")
			extPath, extCleanup, err := testutil.UnzipTemp(tc.extensionPath)
			if err != nil {
				t.Fatalf("Failed to unzip extension: %v", err)
			}
			defer extCleanup()

			t.Log("Configuring extension in Chrome")
			chromeCaps := chrome.Capabilities{
				Path: chromePath,
				Args: []string{
					"--no-sandbox",
					// Specific headless mode that supports extensions. See:
					//   https://bugs.chromium.org/p/chromium/issues/detail?id=706008#c36
					"--headless=chrome",
				},
			}
			if err = chromeCaps.AddUnpackedExtension(extPath); err != nil {
				t.Fatalf("failed to add extension: %v", err)
			}
			caps.AddChrome(chromeCaps)

			t.Log("Starting WebDriver")
			wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
			if err != nil {
				defer dumpLog(t, "SeleniumOutput", &selOut) // Selenium failed to initialize; show debug info.
				t.Fatalf("Failed to start webdriver: %v", err)
			}
			defer func() {
				if qerr := wd.Quit(); qerr != nil {
					t.Errorf("failed to quit webdriver: %v", qerr)
				}
			}()
			defer dumpSeleniumLogs(t, wd)

			t.Log("Navigating to test page")
			path := makeExtensionURL(tc.extensionID, "html/options.html", "test")
			if err = wd.Get(path.String()); err != nil {
				t.Fatalf("Failed to navigate to %s: %v", path, err)
			}

			t.Log("Waiting for navigation")
			if err = wd.WaitWithTimeout(currentURLIs(path.String()), 10*time.Second); err != nil {
				t.Fatalf("Failed to complete navigation to page: %v", err)
			}

			t.Log("Waiting for results")
			if err = wd.WaitWithTimeout(elementExists("failureCount"), 30*time.Second); err != nil {
				t.Fatalf("failed to wait for failure count: %v", err)
			}
			if err = wd.WaitWithTimeout(elementExists("failures"), 30*time.Second); err != nil {
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
		})
	}
}
