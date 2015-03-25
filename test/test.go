package main

import (
	"fmt"
	git "github.com/schani/gogit"
	"os"
)

func test() error {
	r, err := git.Repository("")
	if err != nil {
		return err
	}

	name, err := r.RevParseAbbrev("HEAD")
	if err != nil {
		return err
	}

	oid, err := r.RevParse(name)
	if err != nil {
		return err
	}

	fmt.Printf("%s: %s\n", name, oid)

	parents, err := r.Parents(oid)
	if err != nil {
		return err
	}

	fmt.Printf("parents: %v\n", parents)

	ss, err := r.Status()
	if err != nil {
		return err
	}

	fmt.Printf("Status: %v\n", ss)

	return nil
}

func main() {
	err := test()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
