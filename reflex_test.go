package main

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"
)

// TestMain drains the stdout channel so that informational messages emitted
// during tests do not block on a full channel.
func TestMain(m *testing.M) {
	go func() {
		for range stdout {
		}
	}()
	os.Exit(m.Run())
}

func newTestReflex(command ...string) *Reflex {
	return &Reflex{
		id:        0,
		command:   command,
		subSymbol: "{}",
		timeout:   20 * time.Millisecond,
		mu:        &sync.Mutex{},
	}
}

// Regression for #105: after a failed service start, running can be true while
// cmd has a nil Process. terminate must not panic and must clear running.
func TestTerminateNilSafeAfterFailedStart(t *testing.T) {
	r := newTestReflex("true")
	r.running = true

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.terminate()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("terminate hung")
	}

	if r.Running() {
		t.Fatal("expected running=false after terminate")
	}
}

// A failed start must leave the reflex in a consistent state so that the next
// terminate does not dereference a nil Process.
func TestRunCommandFailedStartClearsRunning(t *testing.T) {
	r := newTestReflex("definitely-not-a-real-reflex-binary-105")

	done, err := r.runCommand("", make(chan OutMsg, 4))
	if err == nil {
		t.Fatal("expected an error for a nonexistent command")
	}
	if done != nil {
		t.Fatal("expected no done channel after a failed start")
	}
	if r.Running() {
		t.Fatal("expected running=false after failed start")
	}
	if r.Killed() {
		t.Fatal("expected killed=false after failed start")
	}
}

// With --sequential, a failed start must release the global command lock,
// otherwise every subsequent command blocks forever.
func TestSequentialUnlockOnFailedStart(t *testing.T) {
	oldSequential := flagSequential
	flagSequential = true
	defer func() { flagSequential = oldSequential }()

	r := newTestReflex("definitely-not-a-real-reflex-binary-105")

	for i := 0; i < 2; i++ {
		done := make(chan struct{})
		go func() {
			defer close(done)
			r.runCommand("", make(chan OutMsg, 4))
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runCommand blocked on the sequential lock after a failed start")
		}
	}
}

// A non-service command that fails to start must not leave runEach waiting on
// a done signal that will never arrive.
func TestRunEachDoesNotHangAfterFailedStart(t *testing.T) {
	r := newTestReflex("definitely-not-a-real-reflex-binary-105")

	names := make(chan string, 2)
	names <- "a"
	names <- "b"
	close(names)

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		r.runEach(names)
	}()

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("runEach hung after a failed start")
	}
}

// killed is per-process: terminating a service must not suppress error output
// from later runs of the same reflex.

func TestKilledResetsBetweenRuns(t *testing.T) {
	r := newTestReflex("sh", "-c", "sleep 5")

	done, err := r.runCommand("", make(chan OutMsg, 16))
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}

	terminateDone := make(chan struct{})
	go func() {
		defer close(terminateDone)
		r.terminate()
	}()

	select {
	case <-terminateDone:
	case <-time.After(3 * time.Second):
		t.Fatal("terminate hung")
	}
	if !r.Killed() {
		t.Fatal("expected killed=true after terminate")
	}
	if r.Running() {
		t.Fatal("expected running=false after terminate")
	}

	// A new run must reset killed so that failures are reported again.
	r.command = []string{"sh", "-c", "exit 0"}
	done, err = r.runCommand("", make(chan OutMsg, 16))
	if err != nil {
		t.Fatalf("second runCommand: %v", err)
	}
	<-done
	if r.Killed() {
		t.Fatal("expected killed=false at the start of a new run")
	}
	if r.Running() {
		t.Fatal("expected running=false after the command exited")
	}
}

func TestPrintMsgTags(t *testing.T) {
	oldDecoration := decoration
	decoration = DecorationPlain
	defer func() { decoration = oldDecoration }()

	for _, tt := range []struct {
		name string
		msg  OutMsg
		want string
	}{
		{"numeric", OutMsg{reflexID: 3, msg: "hello"}, "[03] hello\n"},
		{"named", OutMsg{reflexID: 3, reflexName: "web", msg: "hello"}, "[web] hello\n"},
		{"info", OutMsg{reflexID: -1, msg: "oops"}, "[info] oops\n"},
	} {
		var b bytes.Buffer
		printMsg(tt.msg, &b)
		if got := b.String(); got != tt.want {
			t.Errorf("%s: got %q; want %q", tt.name, got, tt.want)
		}
	}
}
