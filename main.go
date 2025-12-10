package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	name string
	args []string
}

func parseInput(input string) ([]Command, error) {
	// remove empty spaces from the input
	input = strings.TrimSpace(input)
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

/*
*
function execute commands using computer os
*/
func executeNotBuiltInCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

/*
*
function to execute "cd" as a built-in command
*/
func executeCdCommand(args []string) error {
	var path string
	if len(args) == 0 {
		path = os.Getenv("HOME")
	} else {
		path = args[0]
	}
	return os.Chdir(path)
}

/*
*
function to execute single command. Built-in commands cannot be part of pipes
*/
func executeSingleCommand(command Command) error {
	switch command.name {
	case "":
		return nil
	case "exit":
		os.Exit(0)
		return nil
	case "cd":
		return executeCdCommand(command.args)
	default:
		return executeNotBuiltInCommand(command.name, command.args)
	}
}

/*
*
Function to execute a piped command
*/
func executePipeline(commands []Command) error {
	if len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		return executeSingleCommand(commands[0])
	}
	// Check for built-in commands in pipeline
	for _, cmd := range commands {
		if cmd.name == "cd" || cmd.name == "exit" {
			return fmt.Errorf("cannot use built-in command '%s' in pipeline", cmd.name)
		}
	}

	// crete commands
	var cmds []*exec.Cmd //slice of pointers to exec.Cmd so we can modify them later
	for _, command := range commands {
		cmd := exec.Command(command.name, command.args...)
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
			return err
		}
	}

	return nil
}

func main() {

	// read a line of input from the user
	reader := bufio.NewReader(os.Stdin)

	for {
		// display the prompt
		fmt.Print("> ")
		// read the keyboard string
		input, _ := reader.ReadString('\n')

		commands, err := parseInput(input)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if err := executePipeline(commands); err != nil {
			fmt.Println(err)
		}
	}

}
