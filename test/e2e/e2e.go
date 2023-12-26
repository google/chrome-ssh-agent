package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/google/chrome-ssh-agent/go/testutil"
)

var (
	chromePath = testutil.MustRunfile("_main~chromium_dependencies~chromium/chrome-linux/chrome")
)

type LogLevel int

const (
	LogDebug = iota
	LogInfo
	LogWarning
	LogError
	LogFatal
)

func doLog(t *testing.T, level LogLevel, ts time.Time, kind string, m string, args ...any) {
	f := t.Logf
	switch level {
	case LogError:
		f = t.Errorf
	case LogFatal:
		f = t.Fatalf
	}

	prefix := fmt.Sprintf("[%s %s] ", ts.Format(time.RFC3339Nano), kind)
	f(prefix+m, args...)
}

func makeLogFunc(t *testing.T, level LogLevel, kind string) func(string, ...any) {
	return func(m string, args ...any) {
		doLog(t, LogInfo, time.Now(), kind, m, args...)
	}
}

func logConsole(t *testing.T, ev *runtime.EventConsoleAPICalled) {
	var msg bytes.Buffer
	for _, a := range ev.Args {
		msg.Write(a.Value)
		msg.WriteRune(' ')
	}

	switch ev.Type {
	case runtime.APITypeDebug:
		doLog(t, LogDebug, ev.Timestamp.Time(), "Console", msg.String())
	case runtime.APITypeLog:
		doLog(t, LogInfo, ev.Timestamp.Time(), "Console", msg.String())
	case runtime.APITypeWarning:
		doLog(t, LogInfo, ev.Timestamp.Time(), "Console", msg.String())
	case runtime.APITypeError:
		doLog(t, LogError, ev.Timestamp.Time(), "Console", msg.String())
	default:
		doLog(t, LogInfo, ev.Timestamp.Time(), "Console", msg.String())
	}
}

func logException(t *testing.T, ev *runtime.EventExceptionThrown) {
	doLog(t, LogError, ev.Timestamp.Time(), "Console:Exception", ev.ExceptionDetails.Text)
}

type logWriter struct {
	t     *testing.T
	level LogLevel
	kind  string

	w    *io.PipeWriter
	r    *io.PipeReader
	scan *bufio.Scanner

	done chan error
}

func newLogWriter(t *testing.T, level LogLevel, kind string) *logWriter {
	r, w := io.Pipe()
	l := &logWriter{
		t:     t,
		level: level,
		kind:  kind,
		w:     w,
		r:     r,
		scan:  bufio.NewScanner(r),
		done:  make(chan error),
	}
	go l.writeLogs()
	return l
}

func (l *logWriter) Close() error {
	l.w.Close()
	return <-l.done
}

func (l *logWriter) writeLogs() {
	for l.scan.Scan() {
		doLog(l.t, l.level, time.Now(), l.kind, l.scan.Text())
	}
	l.done <- l.scan.Err()
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	return l.w.Write(p)
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

			t.Log("Preparing extension")
			extPath, extCleanup, err := testutil.UnzipTemp(tc.extensionPath)
			if err != nil {
				t.Fatalf("Failed to unzip extension: %v", err)
			}
			defer extCleanup()

			execLogger := newLogWriter(t, LogInfo, "Process")
			defer execLogger.Close()

			t.Log("Initializing Chrome")
			chromeOpts := append(
				chromedp.DefaultExecAllocatorOptions[:],
				chromedp.CombinedOutput(execLogger),
				chromedp.ExecPath(chromePath),
				// Specific headless mode that supports extensions. See:
				//   https://bugs.chromium.org/p/chromium/issues/detail?id=706008#c36
				//   https://bugs.chromium.org/p/chromium/issues/detail?id=706008#c42
				chromedp.Flag("headless", "new"),
				chromedp.Flag("disable-extensions-except", extPath),
				// https://chromium.googlesource.com/chromium/src/+/lkgr/docs/linux/debugging.md#logging
				chromedp.Flag("enable-logging", "stderr"),
				chromedp.Flag("log-level", "1"),
				chromedp.Flag("vlog", "0"),
			)

			actx, acancel := chromedp.NewExecAllocator(
				context.Background(),
				chromeOpts...,
			)
			defer acancel()

			cctx, ccancel := chromedp.NewContext(
				actx,
				chromedp.WithLogf(makeLogFunc(t, LogInfo, "Browser")),
				chromedp.WithErrorf(makeLogFunc(t, LogError, "Browser")),
			)
			defer ccancel()

			chromedp.ListenTarget(cctx, func(ev any) {
				switch ev := ev.(type) {
				case *runtime.EventConsoleAPICalled:
					logConsole(t, ev)
				case *runtime.EventExceptionThrown:
					logException(t, ev)
				}
			})

			ctx, cancel := context.WithTimeout(cctx, 15*time.Second)
			defer cancel()

			t.Log("Running test")
			extURL := makeExtensionURL(tc.extensionID, "html/options.html", "test")
			var failureCountTxt, failures string
			err = chromedp.Run(ctx,
				chromedp.Navigate(extURL.String()),
				chromedp.WaitReady("#failureCount"),
				chromedp.WaitReady("#failures"),
				chromedp.Text("#failureCount", &failureCountTxt),
				chromedp.Text("#failures", &failures),
			)
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}

			failureCount, err := strconv.Atoi(failureCountTxt)
			if err != nil {
				t.Fatalf("Failed to parse failure count '%s' as integer: %v", failureCountTxt, err)
			}

			if failureCount != 0 {
				t.Errorf("Reported Failures:\n%s", failures)
			}
		})
	}
}
