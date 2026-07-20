# ken

Terminal-native spaced-repetition study harness. Markdown flashcards with Bayesian confidence tracking. Your content stays read-only.

## Install

### Pre-built binaries (recommended)

Download the latest release for your platform:

| Platform | Download |
|----------|----------|
| Linux (x86_64) | [ken-linux-amd64](https://github.com/diamondBelema/ken/releases/latest/download/ken-linux-amd64) |
| Windows (x86_64) | [ken-windows-amd64.exe](https://github.com/diamondBelema/ken/releases/latest/download/ken-windows-amd64.exe) |

```bash
# Linux — download, make executable, move to PATH
curl -LO https://github.com/diamondBelema/ken/releases/latest/download/ken-linux-amd64
chmod +x ken-linux-amd64 && mv ken-linux-amd64 ~/.local/bin/ken

# Windows — move ken.exe to a directory in your PATH
```

### From source

```bash
git clone https://github.com/diamondBelema/ken.git
cd ken
go build -o ken ./cmd/ken
mv ken ~/.local/bin/   # or anywhere on $PATH
```

Requires Go 1.21+.

### Install study packages

```bash
ken search nucleic        # search the registry
ken add diamondBelema/nucleic-acid   # install a package
ken list                  # show installed packages
```

## Quick start

```
~/Documents/learn/subjects/
  biochemistry/
    concepts/
      glycolysis.md
    flashcards/
      glycolysis.md
    quizzes/
      glycolysis.md
```

```bash
ken                       # dashboard — confidence spread per subject
ken flashcards biochemistry   # study flashcards (5-level grading)
ken quiz biochemistry         # take a quiz
ken progress biochemistry     # per-concept confidence breakdown
ken notes biochemistry        # manage notes
ken summaries biochemistry    # manage summaries
ken read biochemistry         # read lecture content
ken lint                      # validate all content
```

## Commands

| Command | Description |
|---------|-------------|
| `ken` | Dashboard — confidence spread, due counts, streaks |
| `ken subjects` | List subjects with file counts |
| `ken flashcards <subject>` | Study flashcards (Unknown → Mastered, 5 levels) |
| `ken quiz <subject>` | Quiz mode (MCQ, true/false, fill-in-the-blank) |
| `ken progress [subject]` | Per-concept confidence breakdown |
| `ken stats` | Aggregate statistics |
| `ken notes <subject>` | Manage notes (create, edit, delete, search) |
| `ken summaries <subject>` | Manage summaries |
| `ken read <subject>` | Read lecture notes and content |
| `ken lint [subject]` | Validate content files |
| `ken search <query>` | Search the package registry |
| `ken add <package>` | Install a study package |
| `ken list` | List installed packages |
| `ken remove <package>` | Uninstall a package |
| `ken package` | Generate manifest from local content |
| `ken publish` | Publish packages to the registry |

## How it works

1. **Write Markdown with YAML frontmatter.** Concepts, flashcards, quizzes — plain `.md` files in a folder structure. No editor required, no build step.

2. **ken reads the folder, never writes to it.** Content at `~/Documents/learn/subjects/` stays read-only. Your git repo stays clean.

3. **Confidence updates in your state dir.** Every interaction feeds a Bayesian model with time-decay. Mastery lives on concepts, not cards.

## Content format

```markdown
---
format_version: 1
type: concept_set
set: Glycolysis
concepts:
  - id: c-glycolysis
    name: Glycolysis
    parent_id: null
---

## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate.
```

See [CONTENT-CREATION.md](CONTENT-CREATION.md) for the full format spec, YAML quoting rules, and templates.

## Platform support

| Platform | State dir | File opening |
|----------|-----------|--------------|
| Linux | `~/.local/share/ken/` | `xdg-open` |
| Windows | `~/AppData/Local/ken/` | `cmd /c start` |

## License

MIT
