# ken

Terminal-native learning system for serious study.  
Build your own curriculum in Markdown, learn in focused TUI flows, and track mastery with Bayesian confidence.

![ken logo](https://github.com/user-attachments/assets/e614d980-3bb2-450f-aee0-77a9c12008b9)

## Why ken

- **Three-layer learning flow:** `ken map` → `ken reflect` → `ken read` → `ken flashcards` / `ken quiz`
- **Mastery on concepts, not cards:** Bayesian confidence with decay and anomaly tolerance
- **Content stays read-only:** your study repo is never mutated by progress writes
- **Package marketplace included:** discover, install, update, and publish subject packs
- **Built for keyboard-first focus:** terminal UI powered by Bubble Tea

## Screenshots

![ken read view](https://github.com/user-attachments/assets/58c41a77-7e86-4bce-a1c4-6ca5b2482cdc)

![ken screenshot 2](https://github.com/user-attachments/assets/055faf5c-40f9-4f12-9ea8-413c077d75bd)

![ken screenshot 3](https://github.com/user-attachments/assets/a9de388d-5300-4612-ac3f-d6cac6a8efdd)

## Current project state

All five planned phases are complete.  
Implemented and shipping:

- Core TUI dashboard and study flows
- Familiarity map (`ken map`)
- Reflection layer (`ken reflect`)
- Deep reading with concept hopping (`ken read`)
- Notes system with tags, titles, and markdown rendering
- Flashcards + quiz mastery flows (5-grade scale)
- Bayesian confidence engine with test coverage
- Diagram support (external SVG + Mermaid)
- Course groups (`groups.yaml`)
- Registry/marketplace (`search`, `add`, `list`, `remove`, `package`, `publish`)
- Updaters (`ken update`, `ken self-update`)
- Cross-platform support (Linux + Windows)
- File-locking for safe concurrent package installs

## Install

### Pre-built binaries (recommended)

| Platform | Download |
|----------|----------|
| Linux (x86_64) | [ken-linux-amd64](https://github.com/diamondBelema/ken/releases/latest/download/ken-linux-amd64) |
| Windows (x86_64) | [ken-windows-amd64.exe](https://github.com/diamondBelema/ken/releases/latest/download/ken-windows-amd64.exe) |

```bash
# Linux
curl -LO https://github.com/diamondBelema/ken/releases/latest/download/ken-linux-amd64
chmod +x ken-linux-amd64 && mv ken-linux-amd64 ~/.local/bin/ken
```

### From source

```bash
git clone https://github.com/diamondBelema/ken.git
cd ken
go build -o ken ./cmd/ken
mv ken ~/.local/bin/
```

Requires Go 1.21+.

## Quick start

```text
~/Documents/learn/subjects/
  biochemistry/
    groups.yaml
    concepts/
    flashcards/
    quizzes/
    notes/
```

```bash
ken
ken map biochemistry
ken reflect biochemistry
ken read biochemistry
ken flashcards biochemistry
ken quiz biochemistry
ken notes biochemistry
ken lint biochemistry
```

## Commands

| Command | Description |
|---------|-------------|
| `ken` | Dashboard |
| `ken map <subject>` | Familiarity layer |
| `ken reflect <subject>` | Reflection layer |
| `ken read <subject>` | Deep reading layer |
| `ken flashcards <subject>` | Mastery with 5-level grading |
| `ken quiz <subject>` | Quiz mastery mode |
| `ken notes <subject>` | Notes management |
| `ken lint [subject]` | Content validation |
| `ken search <query>` | Search package registry |
| `ken add <author/package>` | Install package |
| `ken list` | List installed packages |
| `ken remove <author/package>` | Remove package |
| `ken update [package]` | Update packages |
| `ken package` | Generate `ken.yaml` manifest |
| `ken publish` | Publish package to registry |
| `ken version` | Show version/platform |
| `ken self-update` | Update ken binary |

## Architecture at a glance

- **Content (read-only):** `~/Documents/learn/subjects/<subject>/`
- **State (writable):** `~/.local/share/ken/` (Linux) / `~/AppData/Local/ken/` (Windows)
- **Rule:** ken never writes study progress into your content directory

See [CONTENT-CREATION.md](CONTENT-CREATION.md) for full content format details.

## Documentation & website

- Home: https://diamondbelema.github.io/ken/
- Docs: https://diamondbelema.github.io/ken/docs.html

## License

MIT
