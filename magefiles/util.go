//go:build mage

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"
)

type byName []*cobra.Command

type CommandContent struct {
	Name    string
	Content string
}

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }

var coverageFlags = []string{"-coverpkg=./...", "-covermode=atomic", "-coverprofile=coverage.txt"}

// run go test in the root
func goTest(ctx context.Context, path string, args ...string) error {
	return goDirTest(ctx, ".", path, args...)
}

// run go test in a directory
func goDirTest(ctx context.Context, dir string, path string, args ...string) error {
	testArgs, err := testWithArgs(ctx, args...)
	if err != nil {
		return err
	}
	return RunSh(goCmdForTests(), WithV(), WithDir(dir), WithArgs(testArgs...))(path)
}

func goDirTestWithEnv(ctx context.Context, dir string, path string, env map[string]string, args ...string) error {
	testArgs, err := testWithArgs(ctx, args...)
	if err != nil {
		return err
	}
	return RunSh(goCmdForTests(), WithV(), WithDir(dir), WithEnv(env), WithArgs(testArgs...))(path)
}

func testWithArgs(ctx context.Context, args ...string) ([]string, error) {
	testArgs := append([]string{
		"test",
		"-failfast",
		"-count=1",
	}, args...)

	return testArgs, nil
}

// check if docker is installed and running
func checkDocker() error {
	if !hasBinary("docker") {
		return fmt.Errorf("docker must be installed to run e2e tests")
	}
	err := sh.Run("docker", "ps")
	if err == nil || sh.ExitStatus(err) == 0 {
		return nil
	}
	return err
}

// check if a binary exists
func hasBinary(binaryName string) bool {
	_, err := exec.LookPath(binaryName)
	return err == nil
}

// use `richgo` for running tests if it's available
func goCmdForTests() string {
	if hasBinary("richgo") {
		return "richgo"
	}
	return "go"
}

// runOptions is a set of options to be applied with ExecSh.
type runOptions struct {
	cmd            string
	args           []string
	dir            string
	env            map[string]string
	stderr, stdout io.Writer
}

// RunOpt applies an option to a runOptions set.
type RunOpt func(*runOptions)

// WithV sets stderr and stdout the standard streams
func WithV() RunOpt {
	return func(options *runOptions) {
		options.stdout = os.Stdout
		options.stderr = os.Stderr
	}
}

// WithEnv sets the env passed in env vars.
func WithEnv(env map[string]string) RunOpt {
	return func(options *runOptions) {
		if options.env == nil {
			options.env = make(map[string]string)
		}
		for k, v := range env {
			options.env[k] = v
		}
	}
}

// WithStderr sets the stderr stream.
func WithStderr(w io.Writer) RunOpt {
	return func(options *runOptions) {
		options.stderr = w
	}
}

// WithStdout sets the stdout stream.
func WithStdout(w io.Writer) RunOpt {
	return func(options *runOptions) {
		options.stdout = w
	}
}

// WithDir sets the working directory for the command.
func WithDir(dir string) RunOpt {
	return func(options *runOptions) {
		options.dir = dir
	}
}

// WithArgs appends command arguments.
func WithArgs(args ...string) RunOpt {
	return func(options *runOptions) {
		if options.args == nil {
			options.args = make([]string, 0, len(args))
		}
		options.args = append(options.args, args...)
	}
}

func Tool() RunOpt {
	return func(options *runOptions) {
		WithDir("magefiles")(options)
		WithV()(options)
	}
}

// RunSh returns a function that calls ExecSh, only returning errors.
func RunSh(cmd string, options ...RunOpt) func(args ...string) error {
	run := ExecSh(cmd, options...)
	return func(args ...string) error {
		_, err := run(args...)
		return err
	}
}

// ExecSh returns a function that executes the command, piping its stdout and
// stderr according to the config options. If the command fails, it will return
// an error that, if returned from a target or mg.Deps call, will cause mage to
// exit with the same code as the command failed with.
//
// ExecSh takes a variable list of RunOpt objects to configure how the command
// is executed. See RunOpt docs for more details.
//
// Env vars configured on the command override the current environment variables
// set (which are also passed to the command). The cmd and args may include
// references to environment variables in $FOO format, in which case these will be
// expanded before the command is run.
//
// Ran reports if the command ran (rather than was not found or not executable).
// Code reports the exit code the command returned if it ran. If err == nil, ran
// is always true and code is always 0.
func ExecSh(cmd string, options ...RunOpt) func(args ...string) (bool, error) {
	opts := runOptions{
		cmd: cmd,
	}
	for _, o := range options {
		o(&opts)
	}

	if opts.stdout == nil && mg.Verbose() {
		opts.stdout = os.Stdout
	}

	return func(args ...string) (bool, error) {
		expand := func(s string) string {
			s2, ok := opts.env[s]
			if ok {
				return s2
			}
			return os.Getenv(s)
		}
		cmd = os.Expand(cmd, expand)
		finalArgs := append(opts.args, args...)
		for i := range finalArgs {
			finalArgs[i] = os.Expand(finalArgs[i], expand)
		}
		ran, code, err := run(opts.dir, opts.env, opts.stdout, opts.stderr, cmd, finalArgs...)

		if err == nil {
			return ran, nil
		}
		if ran {
			return ran, mg.Fatalf(code, `running "%s %s" failed with exit code %d`, cmd, strings.Join(args, " "), code)
		}
		return ran, fmt.Errorf(`failed to run "%s %s: %v"`, cmd, strings.Join(args, " "), err)
	}
}

func run(dir string, env map[string]string, stdout, stderr io.Writer, cmd string, args ...string) (ran bool, code int, err error) {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, k+"="+v)
	}
	c.Dir = dir
	c.Stderr = stderr
	c.Stdout = stdout
	c.Stdin = os.Stdin

	var quoted []string
	for i := range args {
		quoted = append(quoted, fmt.Sprintf("%q", args[i]))
	}
	// To protect against logging from doing exec in global variables
	if mg.Verbose() {
		log.Println("exec:", cmd, strings.Join(quoted, " "))
	}
	err = c.Run()
	return sh.CmdRan(err), sh.ExitStatus(err), err
}

func GenCustomMarkdownTree(cmd *cobra.Command, dir string) error {
	basename := strings.ReplaceAll(cmd.CommandPath(), " ", "_") + ".md"
	filename := filepath.Join(dir, basename)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return genMarkdownTreeCustom(cmd, f)
}

func genMarkdownTreeCustom(cmd *cobra.Command, f *os.File) error {
	var commandContents []CommandContent

	collectCommandContent(cmd, &commandContents)

	// for sorting commands and their content
	sort.Slice(commandContents, func(i, j int) bool {
		return commandContents[i].Name < commandContents[j].Name
	})

	for _, cc := range commandContents {
		_, err := f.WriteString(cc.Content)
		if err != nil {
			return err
		}
	}

	return nil
}

func collectCommandContent(cmd *cobra.Command, commandContents *[]CommandContent) {
	buf := new(bytes.Buffer)

	name := cmd.CommandPath()

	buf.WriteString("## Reference: `" + name + "`\n\n")
	if len(cmd.Short) > 0 && len(cmd.Long) == 0 {
		buf.WriteString(cmd.Short + "\n\n")
	} else if len(cmd.Short) > 0 {
		buf.WriteString(cmd.Long + "\n\n")
	}

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("### Examples\n\n")
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd); err != nil {
		fmt.Println("Error printing options:", err)
	}

	children := cmd.Commands()
	sort.Sort(byName(children))

	if len(children) > 0 {
		buf.WriteString("### Children commands\n\n")
	}
	for _, child := range children {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}
		cname := name + " " + child.Name()
		link := "reference-" + strings.ReplaceAll(strings.ReplaceAll(cname, "_", "-"), " ", "-")
		buf.WriteString(fmt.Sprintf("- [%s](#%s)\t - %s\n", cname, link, child.Short))
	}
	buf.WriteString("\n\n")

	*commandContents = append(*commandContents, CommandContent{
		Name:    name,
		Content: buf.String(),
	})

	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		collectCommandContent(c, commandContents)
	}
}

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)

	if flags.HasAvailableFlags() {
		buf.WriteString("### Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)

	if parentFlags.HasAvailableFlags() {
		buf.WriteString("### Options Inherited From Parent Flags\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	return nil
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[mK]`)
	return re.ReplaceAllString(s, "")
}

func cleanCommand(cmd *cobra.Command) {
	cmd.Long = stripANSI(cmd.Long)
	cmd.Short = stripANSI(cmd.Short)
	cmd.Example = stripANSI(cmd.Example)

	for _, subCmd := range cmd.Commands() {
		cleanCommand(subCmd)
	}
}
