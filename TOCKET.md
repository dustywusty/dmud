# Tocket Protocol Specification

> How any AI agent should operate in **dmud**.

This file is **agent-agnostic**. Whether you are Claude, Gemini, GPT, Cursor, Cline, Copilot, or any future AI — if you see this file, follow these rules.

---

## 1. Memory Bank

Project context lives in `.context/`, not in chat history. **Read it before doing anything.**

```
.context/
  activeContext.md    <- Start here. Current focus, recent changes, open decisions.
  systemPatterns.md   <- Architecture patterns, conventions, key decisions.
  techContext.md      <- Tech stack, build tools, critical rules.
  productContext.md   <- What the product is, who it's for, why it exists.
  progress.md         <- What's done, what's next.
```

### Rules

- **Read before acting** — Always read `activeContext.md` and `systemPatterns.md` before your first action in a session.
- **Write before leaving** — Update `activeContext.md` with what changed after completing significant work.
- **Trust the files** — If `.context/` says the project uses ESM, it uses ESM. Don't second-guess documented decisions.
- **Don't duplicate** — Context belongs in `.context/`, not scattered in code comments or chat summaries.

---

## 2. Triangulation

Tocket separates **planning** from **implementation** across two agent roles:

```
┌─────────────────┐          ┌─────────────────┐
│    ARCHITECT     │          │    EXECUTOR      │
│  (Planner)       │  payload │  (Implementer)   │
│                  │─────────>│                  │
│  Analyzes task   │          │  Receives plan   │
│  Designs approach│          │  Writes code     │
│  Generates XML   │          │  Runs tests      │
│  Updates patterns│          │  Updates context  │
└─────────────────┘          └─────────────────┘
```

### Architect

- Reads `.context/` to understand current state
- Produces structured payloads (see Section 3) with clear tasks
- Makes architectural decisions and records them in `systemPatterns.md`
- **Does not write code** — only specs and constraints

### Executor

- Reads `.context/` and the Architect's payload
- Implements tasks exactly as specified
- Asks when the plan is unclear — does not improvise architecture
- Updates `activeContext.md` and `progress.md` after completing work

### Solo Mode

Not every task needs triangulation. For simple, well-defined changes, a single agent can act as both Architect and Executor. The Memory Bank rules still apply.

---

## 3. Payloads

A payload is the structured handoff from Architect to Executor.

### Minimal Example

```xml
<payload version="2.0">
  <meta>
    <intent>Goal in one line</intent>
    <scope>Files affected</scope>
    <priority>high | medium | low</priority>
  </meta>
  <tasks>
    <task id="1" type="create | edit | delete">
      <target>file/path</target>
      <action>What to do</action>
      <done>Definition of done</done>
    </task>
  </tasks>
  <validate>
    <check>How to verify success</check>
  </validate>
</payload>
```

---

## Quick Start

1. Read this file (`TOCKET.md`)
2. Read `.context/activeContext.md` for current state
3. Read your role-specific config (`CLAUDE.md` or `GEMINI.md`)
4. Proceed with your task, following the Memory Bank rules above
