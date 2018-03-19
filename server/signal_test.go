// Copyright 2017-2018 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !windows

package server

import (
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	natsd "github.com/nats-io/gnatsd/server"
)

func TestSignalIgnoreUnknown(t *testing.T) {
	opts := GetDefaultOptions()
	opts.HandleSignals = true
	s := runServerWithOpts(t, opts, nil)
	defer s.Shutdown()
	// Send signal that we don't handle
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
	// Check server is still active
	time.Sleep(250 * time.Millisecond)
	if state := s.State(); state != Standalone {
		t.Fatalf("Expected state to be %v, got %v", Standalone.String(), state.String())
	}
}

func TestSignalToReOpenLogFile(t *testing.T) {
	logFile := "test.log"
	defer os.Remove(logFile)
	defer os.Remove(logFile + ".bak")
	sopts := GetDefaultOptions()
	sopts.HandleSignals = true
	nopts := &natsd.Options{
		Host:    "localhost",
		Port:    -1,
		NoSigs:  true,
		LogFile: logFile,
	}
	sopts.EnableLogging = true
	s := runServerWithOpts(t, sopts, nopts)
	defer s.Shutdown()

	// Add a trace
	expectedStr := "This is a Notice"
	s.log.Noticef(expectedStr)
	buf, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	if !strings.Contains(string(buf), expectedStr) {
		t.Fatalf("Expected log to contain %q, got %q", expectedStr, string(buf))
	}
	// Rename the file
	if err := os.Rename(logFile, logFile+".bak"); err != nil {
		t.Fatalf("Unable to rename file: %v", err)
	}
	// This should cause file to be reopened.
	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	// Wait a bit for action to be performed
	time.Sleep(500 * time.Millisecond)
	buf, err = ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	expectedStr = "File log re-opened"
	if !strings.Contains(string(buf), expectedStr) {
		t.Fatalf("Expected log to contain %q, got %q", expectedStr, string(buf))
	}
}
