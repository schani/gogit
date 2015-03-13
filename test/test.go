package main

import (
	"fmt"
	"github.com/schani/git"
	"os"
)

func main() {
	r, err := git.Repository("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ss, err := r.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", ss)
}
