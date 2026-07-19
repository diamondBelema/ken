# Product

## Register

product

## Platform

terminal

## Users

Individual learners studying from markdown-based course material — primarily developers and power users comfortable with CLI tools, markdown, and git, but also any student willing to use the terminal. Users author or import study content as markdown files with YAML frontmatter, then study through a keyboard-driven TUI. AI agents (Claude Code, Codex, etc.) are a secondary user: they write learning content directly to the content folder, creating a seamless agent-to-study pipeline.

## Product Purpose

Ken is a terminal-native spaced-repetition study harness. It reads flashcard, quiz, and concept sets from a folder of markdown files, tracks mastery via a Bayesian confidence algorithm, and presents a fast keyboard-driven TUI. It exists because existing study tools are either bloated GUI applications with cloud lock-in or unstyled CLI tools with no visual hierarchy. Ken fills the gap: fast, focused, and structured — a tool that respects how developers already work while being accessible to any learner.

Success means a user studies consistently, sees their confidence grow on concepts (not just cards), and never loses their content or progress to a service outage or subscription change.

## Positioning

Terminal-native spaced repetition where AI agents write the content and Bayesian confidence tracks your understanding — no cloud, no GUI, no lock-in. Study packages shareable via a GitHub-based registry.

## Brand Personality

Fast, focused, no-nonsense. Like a well-documented library: clear, minimal, gets out of your way. The tone is confident without being loud, structured without being rigid. Ken doesn't explain itself — it shows you what you know and what you don't.

## Anti-references

- **Anki / Quizlet / SaaS flashcard platforms**: GUI-heavy, cloud-dependent, subscription models, bloated interfaces that bury the study loop under settings and sync status
- **Generic CLI tools with walls of text**: Dense unstyled output, no visual hierarchy, overwhelming on first use, no distinction between what matters and what's metadata

## Design Principles

- **Concept-driven mastery**: Cards and quizzes are evidence; understanding lives on concepts. Every interaction updates concept-level Bayesian confidence — the unit of learning is the concept, not the item.
- **Content/state separation**: Content files stay read-only in the user's folder. Progress state lives locally in XDG data dirs. Clean separation always — git repos stay clean, progress survives reinstalls.
- **Agent-friendly by design**: Markdown in, markdown out. AI agents can write learning content directly to the content folder without touching a GUI. The format is simple enough for agents to generate correctly.
- **Shareable via registry**: Study packages can be published to a GitHub-based registry and installed by others with `ken add`. No app store, no account — just a package ID and HTTP.
- **Keyboard-driven speed**: Every action reachable without a mouse. Fast feedback loops, zero wasted keystrokes, vim-style navigation where it makes sense.
- **Non-interrupting workflow**: Notes, summaries, and context switches happen without leaving the study flow. The tool adapts to the learner, not the other way around.

## Accessibility & Inclusion

Minimal — focus on readability. TUI colors chosen for contrast against dark backgrounds. No information conveyed by color alone (grades use text labels alongside color). Keyboard-only input (inherent to TUI). No animation requirements in the terminal interface.
