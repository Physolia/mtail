// Copyright 2011 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

//go:build unix

package tailer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/glog"
	"github.com/google/mtail/internal/logline"
	"github.com/google/mtail/internal/testutil"
	"golang.org/x/sys/unix"
)

// TestTailerOpenRetries is a unix-specific test because on Windows, it is not possible to create a file
// that you yourself cannot read (minimum permissions are 0222).
func TestTailerOpenRetries(t *testing.T) {
	// Can't force a permission denied error if run as root.
	testutil.SkipIfRoot(t)

	ta := makeTestTail(t)

	logfile := filepath.Join(ta.tmpDir, "log")
	if _, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0); err != nil {
		t.Fatal(err)
	}

	testutil.FatalIfErr(t, ta.AddPattern(logfile))

	if err := ta.TailPath(logfile); err == nil || !os.IsPermission(err) {
		t.Fatalf("Expected a permission denied error here: %s", err)
	}
	testutil.FatalIfErr(t, ta.PollLogPatterns())
	testutil.FatalIfErr(t, ta.PollLogStreamsForCompletion())
	glog.Info("remove")
	if err := os.Remove(logfile); err != nil {
		t.Fatal(err)
	}
	testutil.FatalIfErr(t, ta.PollLogPatterns())
	testutil.FatalIfErr(t, ta.PollLogStreamsForCompletion())
	glog.Info("openfile")
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0)
	//nolint:staticcheck // test code
	defer f.Close()
	testutil.FatalIfErr(t, err)
	testutil.FatalIfErr(t, ta.PollLogPatterns())
	testutil.FatalIfErr(t, ta.PollLogStreamsForCompletion())
	glog.Info("chmod")
	if err := os.Chmod(logfile, 0o666); err != nil {
		t.Fatal(err)
	}
	testutil.FatalIfErr(t, ta.PollLogPatterns())
	testutil.FatalIfErr(t, ta.PollLogStreamsForCompletion())
	ta.awaken(1) // force sync to EOF
	glog.Info("write string")
	testutil.WriteString(t, f, "\n")
	ta.awaken(1)

	ta.stop()

	received := testutil.LinesReceived(ta.lines)
	expected := []*logline.LogLine{
		{Context: context.Background(), Filename: logfile, Line: ""},
	}
	testutil.ExpectNoDiff(t, expected, received, testutil.IgnoreFields(logline.LogLine{}, "Context"))
}

func TestAddStdin(t *testing.T) {
	ta := makeTestTail(t)

	name := filepath.Join(ta.tmpDir, "fakestdin")
	testutil.FatalIfErr(t, unix.Mkfifo(name, 0o666))

	oldStdin := os.Stdin
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})

	var err error
	os.Stdin, err = os.OpenFile(name, os.O_RDWR, os.ModeNamedPipe)
	testutil.FatalIfErr(t, err)
	testutil.WriteString(t, os.Stdin, "content\n")

	if err := ta.AddPattern("-"); err != nil {
		t.Errorf("AddPattern(-) -> %v", err)
	}
	testutil.FatalIfErr(t, ta.PollLogPatterns())

	ta.stop()

	received := testutil.LinesReceived(ta.lines)
	expected := []*logline.LogLine{
		{Context: context.Background(), Filename: "-", Line: "content"},
	}
	testutil.ExpectNoDiff(t, expected, received, testutil.IgnoreFields(logline.LogLine{}, "Context"))
}
