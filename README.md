# gocshell

A simple, lightweight shell implementation written in Go.

## Features

- **Command Execution**: Execute any system command available on your PATH
- **Built-in Commands**: Native support for `cd`, `exit`, and `history`
- **Piped Commands**: Chain multiple commands with pipes (`|`)
- **Signal Handling**: Graceful handling of CTRL+C (SIGINT)
- **Command History**: Persistent command history saved to `~/.gocsh_history`

## Installation

### Prerequisites
- Go 1.16 or higher

### Build from source

```bash
git clone <repository-url>
cd gocshell
go build -o gocsh main.go
```

### Run

```bash
./gocsh
```

## Usage

### Basic Commands

```bash
> ls -la
> pwd
> echo "Hello, World!"
```

### Built-in Commands

#### `cd` - Change Directory
```bash
> cd /path/to/directory
> cd # Changes to $HOME directory
```

#### `history` - Display Command History
```bash
> history # Shows all saved commands
```

#### `exit` - Exit the Shell
```bash
> exit
```

### Piped Commands

Chain multiple commands together using pipes:

```bash
> ls -l | grep go
> cat file.txt | wc -l
> ps aux | grep python | wc -l
```

### Signal Handling

Press `CTRL+C` to cancel a running command without exiting the shell.

## Technical Details

## File Structure

```
gocshell/
├── main.go          # Main shell implementation
├── go.mod           # Go module file
└── README.md        # This file
```

## Command History

Command history is stored in `~/.gocsh_history`. The shell automatically:
- Saves all commands (except `history` and `exit`)
- Persists history across sessions
- Opens the history file once on startup for efficiency

## Development

### Running in Development

```bash
go run main.go
```

### Building

```bash
go build -o gocsh main.go
```
