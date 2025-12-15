package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

var history_file = os.Getenv("HOME") + "/.gocsh_history"
var historyFile *os.File

type Command struct {
	name string
	args []string
}

func parseInput(input string) ([]Command, error) {
	// remove empty spaces from the input
	input = strings.TrimSpace(input)

	// return empty command slice if input is empty
	if input == "" {
		return []Command{}, nil
	}

	// split input by |
	pipedInputs := strings.Split(input, "|")
	commands := make([]Command, 0, len(pipedInputs))

	// per each piped command identify command and args
	for _, pipedInput := range pipedInputs {
		pipedInput := strings.TrimSpace(pipedInput)
		parts := strings.Fields(pipedInput)

		if len(parts) == 0 {
			return []Command{}, fmt.Errorf("invalid input: %s", pipedInput)
		}
		command := parts[0]
		args := parts[1:]
		commands = append(commands, Command{command, args})
	}
	return commands, nil
}

func initHistory() error {
	var err error
	historyFile, err = os.OpenFile(history_file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	return err
}

func saveHistory(input string) error {
	if historyFile == nil {
		return nil // History disabled
	}
	_, err := historyFile.WriteString(input)
	return err
}

func closeHistory() {
	if historyFile != nil {
		historyFile.Close()
	}
}

func displayHistory() error {
	file, err := os.Open(history_file)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(os.Stdout, file)
	return err
}

func shouldBeInHistory(commands []Command) bool {
	if len(commands) == 0 {
		return false // Empty commands should not be saved
	}
	if len(commands) > 1 {
		return true // Save piped commands to history
	}
	// Don't save "history" or "exit" commands
	return commands[0].name != "history" && commands[0].name != "exit"

}

// setupSignalHandler creates a context that will be cancelled when CTRL+C is pressed.
// Returns the context and a cleanup function that should be deferred.
func setupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		cancel()
	}()

	// Return a wrapped cancel function that also stops signal notifications
	cleanup := func() {
		signal.Stop(sigChan)
		cancel()
	}

	return ctx, cleanup
}

// handleCommandError checks if an error is due to context cancellation (CTRL+C).
// If so, it prints a newline and returns nil. Otherwise, it returns the original error.
func handleCommandError(ctx context.Context, err error) error {
	if err != nil && errors.Is(ctx.Err(), context.Canceled) {
		fmt.Println() // Print newline after ^C
		return nil    // Don't treat ^C as an error
	}
	return err
}

// executeNotBuiltInCommand executes commands using the computer's OS.
func executeNotBuiltInCommand(command string, args []string) error {
	ctx, cleanup := setupSignalHandler()
	defer cleanup()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	return handleCommandError(ctx, err)
}

// executeCdCommand executes the "cd" built-in command to change directories.
func executeCdCommand(args []string) error {
	var path string
	if len(args) == 0 { // if no path is defined it defaults to $HOME
		path = os.Getenv("HOME")
	} else {
		path = args[0]
	}
	return os.Chdir(path)
}

// executeSingleCommand executes a single command. Built-in commands cannot be part of pipes.
func executeSingleCommand(command Command) error {
	switch command.name {
	case "":
		return nil
	case "exit":
		os.Exit(0)
		return nil
	case "cd":
		return executeCdCommand(command.args)
	case "history":
		return displayHistory()
	default:
		return executeNotBuiltInCommand(command.name, command.args)
	}
}

// executePipeline executes a series of piped commands.
func executePipeline(commands []Command) error {
	if len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		return executeSingleCommand(commands[0])
	}

	// Create context so it can be cancelled
	ctx, cleanup := setupSignalHandler()
	defer cleanup()

	// Check for built-in commands in pipeline
	for _, cmd := range commands {
		if cmd.name == "cd" || cmd.name == "exit" {
			return fmt.Errorf("cannot use built-in command '%s' in pipeline", cmd.name)
		}
	}

	// create commands
	var cmds []*exec.Cmd //slice of pointers to exec.Cmd so we can modify them later
	for _, command := range commands {
		cmd := exec.CommandContext(ctx, command.name, command.args...)
		cmds = append(cmds, cmd)
	}

	// Connect the output of each command to the input of the next command
	// the last command has no "next" command to connect to
	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			return err
		}
		cmds[i+1].Stdin = stdout
	}

	// Set first command stdin and last command stdout to the terminal
	cmds[0].Stdin = os.Stdin
	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[len(cmds)-1].Stderr = os.Stderr

	// Start all commands
	// we use use Start(non-blocking) instead of Run(blocking), we need all cmds running in parallel so next command
	// can read from the prev pipe
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return err
		}
	}

	// Wait for all commands
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return handleCommandError(ctx, err)
		}
	}

	return nil
}

func main() {
	// Initialize history file
	if err := initHistory(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open history: %v\n", err)
	}
	defer closeHistory()

	// read a line of input from the user
	reader := bufio.NewReader(os.Stdin)

	for {
		// display the prompt
		fmt.Print("> ")
		// read the keyboard string
		input, _ := reader.ReadString('\n')

		// parse input to Command
		commands, err := parseInput(input)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if err := executePipeline(commands); err != nil {
			fmt.Println(err)
		}

		// save to history
		if shouldBeInHistory(commands) {
			if err := saveHistory(input); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write to history: %v\n", err)
			}
		}
	}

}
