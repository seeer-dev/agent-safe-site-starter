package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var defaultSteps = [][]string{
	{"go", "run", "./server/tools/archcheck"},
	{"go", "run", "./server/tools/migration-parity"},
	{"go", "run", "./server/tools/speccheck"},
	{"go", "run", "./server/tools/scopecheck"},
	{"go", "test", "./..."},
	{"go", "test", "./server/internal/modules/commerce", "./server/internal/modules/staff", "./server/internal/modules/media", "-count=10"},
	{"go", "vet", "./..."},
}

func main() {
	for _, step := range defaultSteps {
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
