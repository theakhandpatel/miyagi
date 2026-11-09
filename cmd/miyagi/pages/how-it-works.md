---
description: How Miyagi grades you, why the model is faked, and what each type of check actually proves.
---

# How it works

Most courses tell you to run your program and decide for yourself whether it worked. That
fails for a reason specific to this subject: a model is random and slow, so "it worked
yesterday" tells you nothing about whether *you* broke something.

So the course ships a second program that plays the model.

## The fake provider

`internal/fake` is a real HTTP server speaking real Server-Sent Events on a real socket.
Only the words in the reply are fixed. Your program cannot tell the difference, because
there is no difference: it makes a real request over loopback and parses real frames.

That is what makes grading deterministic, instant, and free. It is also the honest version
of the promise a hosted course makes: nothing about your code is sent anywhere.

## Three kinds of check

The rule is to grade the largest thing that can carry the claim.

**Black box, by default.** Your binary is built and run as a process. The case asserts on
what a user could observe: stdout, the exit code, the files changed, and the requests that
reached the server. Nine chapters out of ten are graded this way, because a harness is a
program and a program should be proven as one.

**A pinned function, rarely.** Only when the claim is a pure transform with awkward inputs,
where forty throwaway processes would buy no realism. `miyagi submit` states the surface up
front when a chapter does this, so you never discover it from a compiler error.

**A recorded transcript, never.** Nothing diffs your terminal output against a golden file.
That would grade formatting rather than behaviour, and it would break on a terminal width.

## What a failure looks like

```
  FAIL 04 omits optional numbers instead of sending zero
        why: Zero is not the same as unset. A body containing "max_tokens": 0 tells a
             real provider to produce nothing, so a harness that always serialises the
             field passes the fixture and fails in production.
        field: request body max_tokens
        expected: absent
        actual:   0
        hint: Leave the field out of the struct with omitempty.
        more: miyagi hint 01 optional-zero-values
```

The `why` is the part that matters. It is the sentence a senior engineer would say if they
were looking over your shoulder, and it is why the course can be taken without one.

## The build is not a failure

A build error is reported as "not ready", never as a failed case. A learner who mistypes an
import should not see a wall of compiler output and conclude their logic is wrong. The
grader triages build failures into a single sentence with a pointer.

## Prior chapters stay green

Chapters are cumulative, and the reference for each one is derived by subtracting the newer
ones. That is what makes "builds upon the tests" a mechanical fact rather than a hope: when
you submit chapter 12, the checks for chapters 01 to 11 are still there and still run
against the same program.

`miyagi verify` proves the stronger property for the course itself: every chapter's
reference passes every case, and every chapter's before-state fails at least one. A chapter
whose starter already passes grades nothing, and that is treated as a bug in the course.
