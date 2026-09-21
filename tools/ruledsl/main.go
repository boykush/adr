// Command ruledsl holds the rule files to ADE's rule DSL at the version go.mod
// requires: it checks that they parse, and writes the DSL reference that the
// ade-rule-dsl skill in boykush/ai-plugins carries. Both come from one ADE
// build, so the grammar a session reads a rule by is the one it was checked by.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/phi42/ad-enforcement-tool/dsl"
)

const adeModule = "github.com/phi42/ad-enforcement-tool"

const usage = `usage:
  ruledsl validate PATH...  parse each .rule file, or each one directly in a directory
  ruledsl vendor DIR        write ADE's DSL reference and its license into DIR`

const referenceFile = "dsl-reference.md"

// referenceHeader records where the copy below it came from, as the Apache
// License asks of a redistributed file.
const referenceHeader = `<!--
Vendored from %s %s (dsl/dsl-reference.md). Nothing below this comment is changed.
Licensed under the Apache License, Version 2.0: see LICENSE beside this file.
Written by tools/ruledsl in github.com/boykush/adr for the ADE version its go.mod requires. Do not edit.
-->
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	switch {
	case len(args) >= 2 && args[0] == "validate":
		return validate(args[1:], stdout, stderr)
	case len(args) == 2 && args[0] == "vendor":
		return vendor(args[1])
	}
	return errors.New(usage)
}

// validate parses each rule file the way ADE does and stops there. Nothing
// runs a rule: the sessions that read the rules hold their work to them.
func validate(paths []string, stdout, stderr io.Writer) error {
	files, err := ruleFiles(paths)
	if err != nil {
		return err
	}
	failed := 0
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err == nil {
			err = dsl.Validate(string(src))
		}
		if err != nil {
			failed++
			fmt.Fprintf(stderr, "X %s: %v\n", file, err)
			continue
		}
		fmt.Fprintf(stdout, "✓ %s\n", file)
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d rule files are not valid ADE DSL", failed, len(files))
	}
	return nil
}

// ruleFiles expands each directory into the .rule files directly inside it.
// Finding none at all is an error, so a mistyped path cannot pass as clean.
func ruleFiles(paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(path, "*.rule"))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .rule files in %s", strings.Join(paths, ", "))
	}
	return files, nil
}

func vendor(dir string) error {
	version, err := adeVersion()
	if err != nil {
		return err
	}
	license, err := moduleFile("LICENSE")
	if err != nil {
		return err
	}
	reference := fmt.Sprintf(referenceHeader, adeModule, version) + dsl.Reference
	if err := os.WriteFile(filepath.Join(dir, referenceFile), []byte(reference), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "LICENSE"), license, 0o644)
}

// adeVersion is the ADE version linked into this build, which is the one
// go.mod requires and so the one whose parser validate runs.
func adeVersion() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("no build info to read the ADE version from")
	}
	for _, dep := range info.Deps {
		if dep.Path == adeModule {
			return dep.Version, nil
		}
	}
	return "", fmt.Errorf("%s is not in this build", adeModule)
}

// moduleFile reads a file from the ADE sources go.mod requires. The dsl
// package embeds the reference but not the license, so the go command is
// asked where those sources are.
func moduleFile(name string) ([]byte, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", adeModule).Output()
	if err != nil {
		return nil, fmt.Errorf("locating %s: %w", adeModule, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return nil, fmt.Errorf("%s is not downloaded (run: go mod download)", adeModule)
	}
	return os.ReadFile(filepath.Join(dir, name))
}
