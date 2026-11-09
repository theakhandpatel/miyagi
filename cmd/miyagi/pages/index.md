---
layout: home
description: A twenty-one chapter Go course where you build a coding-agent harness, graded offline by a verifier that tells you exactly what is wrong.
hero:
  name: Miyagi
  text: Teach me harness, my inner AI.
  tagline: Twenty-one chapters that take you from a single HTTP request to a working coding agent. Graded offline, by a program that says what is wrong and why.
  image:
    src: /docs/assets/logo.png
    alt: Miyagi
  actions:
    - theme: brand
      text: Start chapter 01
      link: /chapters/01-chat-with-the-model/
    - theme: alt
      text: How it works
      link: /how-it-works
features:
  - title: It grades you
    details: Every chapter has hidden checks. A failure names the field, the expected value, the actual value, and why it matters. No reading your own output and guessing.
  - title: No model, no key, no bill
    details: The verifier plays the model with a real HTTP server speaking real server-sent events. Deterministic, instant, and free to run.
  - title: One program that grows
    details: You keep editing agent/ from chapter 01 to 21. Nothing is thrown away and nothing is re-scaffolded, so the accumulated context is yours.
  - title: You keep the result
    details: MIT, about 4,300 lines you understand, and it runs against any OpenAI-compatible endpoint when you point it at a real model.
---

## The arc

Twenty-one chapters in seven parts. Each one ends with a sentence you can say about the
program you have.

| Part | Chapters | By the end you can |
| --- | --- | --- |
| I Wire | 01 to 02 | make a streaming request, and reassemble tool calls from fragments |
| II Act | 03 to 05 | give the model tools that touch disk, and run the loop that uses them |
| III Remember | 06 to 08 | inject repo instructions, persist sessions, and stay inside a context budget |
| IV Extend | 09 to 11 | interrupt a run, steer it, and load skills on demand |
| V Compose | 12 to 13 | hand a job to a subagent, and call tools in another process |
| VI Survive | 14 to 18 | retry, bound a tool result, cap schemas, scope skills, build the request |
| VII Store | 19 to 21 | keep conversations in SQLite, report cost, and configure a provider |

## Install

```sh
go install github.com/theakhandpatel/miyagi/cmd/miyagi@v0.1.0
```

Then clone the course and run it. The verifier needs no account and no network.

## Why the name

Miyagi is the teacher from The Karate Kid. He never lectures. He hands you a rag and a car
and makes you repeat a motion until it is automatic, then he throws a punch to see whether
it actually is. That is what this course does with code, and it is why the tool is named
after the thing that tests you rather than the thing you read.
