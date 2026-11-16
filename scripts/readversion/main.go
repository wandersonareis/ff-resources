// Command readversion imprime o conteúdo de um arquivo VERSION sem espaços
// nas pontas. É usado pelas Taskfiles para injetar as versões via -ldflags de
// forma portátil (Windows/Linux/macOS), sem depender de `cat`.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: readversion <arquivo>")
		os.Exit(2)
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "readversion: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(strings.TrimSpace(string(b)))
}
