// Copyright 2011 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

// +build gofuzz

package vm

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"

	"github.com/google/mtail/internal/logline"
)

// U+2424 SYMBOL FOR NEWLINE
const SEP = "␤"

// Enable this when debugging with a fuzz crash artifact; it slows the fuzzer down when enabled.
const dumpDebug = false

func Fuzz(data []byte) int {
	offset := bytes.Index(data, []byte(SEP))
	if offset < 0 {
		offset = len(data)
		data = append(data, []byte(SEP)...)
	}
	fmt.Printf("data len %d, offset is %d, input starts at %d\n", len(data), offset, offset+len(SEP))

	v, err := Compile("fuzz", bytes.NewReader(data[:offset]), dumpDebug, dumpDebug, false, nil, 0, 0)
	if err != nil {
		fmt.Println(err)
		return 0 // false
	}
	if dumpDebug {
		fmt.Println(v.DumpByteCode())
	}
	v.HardCrash = true
	scanner := bufio.NewScanner(bytes.NewBuffer(data[offset+len(SEP):]))
	for scanner.Scan() {
		v.ProcessLogLine(context.Background(), logline.New(context.Background(), "fuzz", scanner.Text()))
	}
	return 1
}

func init() {
	// We need to successfully parse flags to initialize the glog logger used
	// by the compiler, but the fuzzer gets called with flags captured by the
	// libfuzzer main, which we don't want to intercept here.
	flag.CommandLine.Parse([]string{})
}
