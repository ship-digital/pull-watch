# pull-watch

```
               _ _                 _       _
              | | |               | |     | |
     _ __  _  | | |  __      ____ | |_ ___| |__
    | '_ \| | | | | \ \ /\ / / _` | __/ __| '_ \
    | |_) | |_| | |  \ V  V / (_| | || (__| | | |
    | .__/ \__,_|__\  \_/\_/ \__,_|\__\___|_|  \_\
    | |
    |_|
```

A hobbyist-grade tool that watches a git repository for changes and runs a specified command when changes are detected.

Your friendly neighborhood Git repository watchdog! 🐕
Guards your watch, your time, avoids wasting it, get it? No? Never mind. I am just a dad. It's muscle memory at this point!

Never miss a change again - pull-watch sits patiently by your repository, waiting to spring into action whenever something new comes along. Perfect for developers who want their local environment to stay in sync without the hassle of manual updates.

## ✨ Features

- 🕒 Configurable poll interval (because sometimes you need a coffee break)
- 🎯 Graceful process management (no rough handling here!)
- 📁 Support for different git directories (home is where your .git is)
- 📢 Verbose logging option (for when you're feeling chatty)
- 🛡️ Proper signal handling (catches signals like a pro)
- ⏱️ Context-aware git operations with timeouts (patience is a virtue, but timeouts are better)

## 🚀 Installation

Quick and easy - just like ordering pizza!

```bash
go install github.com/ship-digital/pull-watch@latest
```

## 🎮 Usage

```
pull-watch [options] -- <command>

Options:
  -interval duration
        Poll interval (e.g. 15s, 1m) (default 15s)
  -git-dir string
        Git repository directory (default ".")
  -verbose
        Enable verbose logging
  -graceful
        Try graceful stop before force kill
  -stop-timeout duration
        Timeout for graceful stop before force kill (default 5s)
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

---

Made with ❤️ by [@deblasis](https://github.com/deblasis) for developers who appreciate a touch of automation in their lives.