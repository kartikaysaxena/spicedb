//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/authzed/spicedb/pkg/cmd"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Gen mg.Namespace

// All Run all generators in parallel
func (g Gen) All() error {
	mg.Deps(g.Go, g.Proto, g.Docs)
	return nil
}

// Go Run go codegen
func (Gen) Go() error {
	fmt.Println("generating go")
	return sh.RunV("go", "generate", "./...")
}

// Docs Generate documentation
func (g Gen) Docs() error {
	targetDir := "../docs"

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return err
	}

	rootCmd, err := cmd.BuildRootCommand()
	if err != nil {
		return err
	}

	cleanCommand(rootCmd)
	return GenCustomMarkdownTree(rootCmd, targetDir)
}

// Proto Run proto codegen
func (Gen) Proto() error {
	fmt.Println("generating buf")
	return RunSh("go", Tool())("run", "github.com/bufbuild/buf/cmd/buf",
		"generate", "-o", "../pkg/proto", "../proto/internal", "--template", "../buf.gen.yaml")
}

func (Gen) Completions() error {
	if err := RunSh("mkdir", WithDir("."))("-p", "./completions"); err != nil {
		return err
	}

	for _, shell := range []string{"bash", "zsh", "fish"} {
		f, err := os.Create("./completions/spicedb." + shell)
		if err != nil {
			return err
		}
		defer f.Close()

		if err := RunSh("go", WithDir("."), WithStdout(f))("run", "./cmd/spicedb/main.go", "completion", shell); err != nil {
			return err
		}
	}
	return nil
}
