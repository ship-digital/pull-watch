```
             _ _                      _       _
            | | |                    | |     | |
 _ __  _   _| | |________      ____ _| |_ ___| |__
| '_ \| | | | | |______\ \ /\ / / _` | __/ __| '_ \
| |_) | |_| | | |       \ V  V / (_| | || (__| | | |
| .__/ \__,_|_|_|        \_/\_/ \__,_|\__\___|_| |_|
| |
|_|
```

# pull-watch -- ./do_something_when_your_git_repo_changes.sh

A hobbyist-grade tool that watches a git repository for changes and runs a specified command when changes are detected.

Your friendly neighborhood Git repository watchdog! 🐕
Guards your watch, your time, avoids wasting it, get it? No? Never mind. I am just a dad. It's muscle memory at this point!

![Which way?](./assets/meme_castles.png)

> **How it started**: 🔄 `git pull` && `./run.sh` && CTRL/CMD+C && `git pull` && `./run.sh` && CTRL/CMD+C && ♾️... 😵
>
> **How it's going**: `pull-watch -- ./run.sh`

## ✨ Features

- 🕒 Configurable poll interval (because sometimes you need a coffee break)
- 🎯 Graceful process management (no rough handling here!)
- 📁 Support for different git directories (home is where your .git is)
- 📢 Smart logging levels (quiet, normal, and verbose - you choose how chatty it gets!)
- 🛡️ Proper signal handling (catches signals like a pro)
- ⏱️ Context-aware git operations with timeouts (patience is a virtue, but timeouts are better)
- 🔄 Run on start option (for the eager beavers)
- ⌚ Optional timestamps in logs (when you need to know when things happened)

## 🚀 Installation

Quick and easy - just like ordering pizza!

### Go Install

```bash
go install github.com/ship-digital/pull-watch@latest
```

### Homebrew

```bash
brew tap ship-digital/tap
brew install pull-watch
```

### Chocolatey (Windows)

```powershell
choco install pull-watch
```

### NPM / NPX

You can install it globally:

```bash
npm install -g @ship-digital/pull-watch
```

Or run it directly using npx:

```bash
npx @ship-digital/pull-watch -- <your_command>
```

## 🎮 Usage

```

  Usage: pull-watch [options] -- <command>

   Watch git repository for remote changes and run commands.

   It's like: 'git pull && <command>' but with polling and automatic process management.

  Options:
    -git-dir string
      	Git repository directory (default ".")
    -graceful
      	Try graceful stop before force kill
    -interval duration
      	Poll interval (e.g. 15s, 1m) (default 15s)
    -no-restart
      	Pull changes without restarting the command, useful if the command has a built-in auto-reload feature
    -quiet
      	Show only errors and warnings
    -run-on-start
      	Run command on startup regardless of git state
    -stop-timeout duration
      	Timeout for graceful stop before force kill (default 5s)
    -timestamp
      	Show timestamps in logs
    -verbose
      	Enable verbose logging
    -version
      	Show version information

```

## 🌟 Examples

### Watch current directory and restart a server when changes are detected:

Keep your server fresh and up-to-date!

```bash
pull-watch -- go run main.go
```

### Watch specific directory with custom interval:

For when you want to keep an eye on things from a distance...

```bash
pull-watch -git-dir /path/to/repo -interval 1m -- npm start
```

### Force kill processes (default):

The "no time for chitchat" approach

```bash
pull-watch -- node server.js
```

### Graceful stop before force kill:

For the gentler souls among us

```bash
pull-watch -graceful -stop-timeout 10s -- ./my-server
```

### Watch with different logging levels:

```bash
# Default mode - shows important info
pull-watch -- npm start

# Quiet mode - shows only errors and warnings
pull-watch -quiet -- npm start

# Verbose mode - shows all the details
pull-watch -verbose -- npm start
```

---

Made with ❤️ by [@deblasis](https://github.com/deblasis) for developers who appreciate a touch of automation in their lives.
