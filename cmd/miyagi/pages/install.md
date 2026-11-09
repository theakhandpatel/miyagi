---
description: Install the Miyagi grader and start chapter 01. No account, no API key, no network.
---

# Install

Two commands. The first installs the grader, the second fetches the course and runs it.

## Install the grader

```sh
go install github.com/theakhandpatel/miyagi/cmd/miyagi@v0.1.0
```

That puts `miyagi` in `$(go env GOPATH)/bin`, which needs to be on your `PATH`. If the
command is not found afterwards:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

You need Go 1.26 or later. Nothing else.

## Get the course

```sh
git clone https://github.com/theakhandpatel/miyagi
cd miyagi
```

## Start

```sh
miyagi            # where you are, and what to do next
miyagi read 01    # the task for chapter 01
miyagi submit 01  # compile, grade, report
```

That is the whole loop. Read the chapter, edit `agent/`, submit, read the failures, repeat.

## What you do not need

No account. No API key. No Docker. No network after the clone. The verifier plays the model
with a local HTTP server, so grading is deterministic and costs nothing.

You need a real endpoint only when you point your finished agent at an actual model, which
is chapter 21.

## If something looks broken

```sh
miyagi doctor
```

It checks the things that produce misleading failures: the Go version, whether the entry
point still compiles, a bindable loopback port, a writable scratch directory, and whether
you are in a terminal.

## How the grading actually works

Your program is built once and then run as a real process. The checks never import your
package, so you may restructure `agent/` however you like; if you move the entry point, say
so with `MIYAGI_ENTRY`.

Each case runs the binary against a fake provider that speaks real server-sent events, and
asserts on what crossed the boundary: the request that was sent, the event stream, the
files written, the exit code.
