```markdown
## mlogger — Minimalistic Logger for Go

A lightweight logging package for Go projects. Perfect for learning Go or using in small‑scale applications where you need basic logging functionality without the overhead of large frameworks.

### Key Features

- **Simplicity**: Easy to set up and use — just import and start logging.
- **Small size**: Minimal codebase, no external dependencies.
- **Educational value**: Great for learning how logging systems work under the hood.
- **Flexible output**: Log to console, file, or both.
- **Color‑coded console output**: Visual distinction between log levels (optional).
- **Log rotation**: Automatic file rotation based on size with backup management.
- **Configurable levels**: Support for `DEBUG`, `INFO`, `WARN`, `ERROR` levels.
- **Caller info**: Show file name and line number in logs (optional).
- **Asynchronous logging**: Optional buffered writing for better performance.
- **Time formatting**: Choose between full datetime or just time.

### Installation

```bash
go get github.com/your-username/mlogger
```

### Configuration

Before using the logger, adjust the constants in the package to fit your needs:

```go
const (
	LOG_OUTPUT = "console"        // "file", "console", "both"
	LOG_FILE_PATH = "logs/app.log"
	LOG_LEVEL = "debug"           // "debug", "info", "warn", "error"
	LOG_TIME_FORMAT = "datetime"  // "datetime" or "time"
	LOG_SHOW_CALLER = true
	LOG_COLOR_CONSOLE = true
	LOG_BUFFER_SIZE = 100       // 0 for synchronous writing
	LOG_MAX_SIZE_MB = 10       // 0 for no limit
	LOG_MAX_BACKUPS = 3         // 0 to keep all backups
)
```

### Usage Examples

### Basic Logging

```go
package main

import "github.com/your-username/mlogger"

func main() {
	mlogger.Info("Application started")
	mlogger.Debug("Debugging information: %d items processed", 42)
	mlogger.Warn("Something might go wrong")
	mlogger.Error("An error occurred: %v", err)
	// mlogger.Fatal("Fatal error — program will exit")
}
```

### Dynamic Level Change

```go
mlogger.SetLevel("warn") // Only WARN and ERROR messages will be shown
mlogger.Info("This won't be printed")
mlogger.Warn("This will be printed")
```

### Forced Buffer Sync

```go
// Force flush the buffer to disk
mlogger.Sync()
```

### Graceful Shutdown

Always call `Close()` before your program exits when using asynchronous logging:

```go
func main() {
	defer mlogger.Close() // Ensure all logs are written before exit

	mlogger.Info("Starting application")
	// ... your application logic ...
	mlogger.Info("Shutting down gracefully")
}
```

### Output Format

Example log output (with `LOG_SHOW_CALLER = true` and `LOG_COLOR_CONSOLE = true`):

```
[2024-01-15 10:30:45] [INFO] main.go:42 Application started
[2024-01-15 10:30:46] [DEBUG] utils.go:123 Processing item 42
[2024-01-15 10:30:47] [WARN] service.go:88 Connection timeout
[2024-01-15 10:30:48] [ERROR] db.go:55 Database connection failed
```

### Log Rotation

When the log file reaches `LOG_MAX_SIZE_MB`, it will be rotated:
- Current log file is renamed with a timestamp suffix.
- Old backups are cleaned up to maintain only `LOG_MAX_BACKUPS` files.
- A new log file is created.

### License

MIT License — see [LICENSE](LICENSE) for details.
