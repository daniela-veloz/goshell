package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {

	// read a line of input from the user
	reader := bufio.NewReader(os.Stdin)

	for {
		// display the prompt
		fmt.Print("> ")
		// read the keyboard string
		input, _ := reader.ReadString('\n')
		// remove empty spaces from the input
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// split input
		parts := strings.Fields(input)
		// create commands, first string is the command and the rest are arguments
		command := parts[0]
		args := parts[1:]

		if command == "exit" {
			os.Exit(0)
		}

		cmd := exec.Command(command, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Println(err.Error())
		}
	}

}
