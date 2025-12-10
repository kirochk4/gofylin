package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	yv "github.com/kirochk4/goyeva/yeva"
)

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--help":
			showHelp()
			return
		case "--version":
			fmt.Printf("yeva %s", yv.Version)
			return
		}
	}

	var err error
	if len(os.Args) == 1 {
		err = runRepl()
	} else {
		err = runFile(os.Args[1:])
	}
	if err != nil {
		os.Exit(1)
	}
}

func runFile(args []string) error {
	scriptPath := args[0]
	source, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("run file: %w", err)
	}
	e := yv.New()
	argv := &yv.Doc{Pairs: make(map[yv.Value]yv.Value, len(args))}
	for i, arg := range args {
		argv.Pairs[yv.Num(i)] = yv.Str(arg)
	}
	e.Globals["argv"] = argv
	return e.Interpret(source)
}

func runRepl() error {
	vm := yv.New()
	fmt.Printf("Yeva %s\n", yv.Version)
	fmt.Println("exit using ctrl+c")
	for {
		fmt.Print(">>> ")
		source, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("run repl: %w", err)
		}
		source = source[:len(source)-1] // remove delim
		vm.Interpret(source)
	}
}

func showHelp() {
	fmt.Printf("Yeva %s\n", yv.Version)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println(format("repl", "yeva"))
	fmt.Println(format("file", "yeva [file] [arguments...]"))
	fmt.Println()
	fmt.Println("Optional arguments:")
	fmt.Println(format("--help", "Show command line usage"))
	fmt.Println(format("--version", "Show version"))
}

func format(arg, desc string) string {
	return fmt.Sprintf("  %-18s%s", arg, desc)
}
