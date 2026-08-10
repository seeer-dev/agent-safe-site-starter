package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	steps := [][]string{
		{"go", "run", "./server/tools/archcheck"},
		{"go", "run", "./server/tools/scopecheck"},
		{"go", "test", "./..."},
		{"go", "vet", "./..."},
	}
	for _, step := range steps {
		fmt.Println("verify:", strings.Join(step, " "))
		cmd := exec.Command(step[0], step[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
	}
	fmt.Println("verify: ok")
}
