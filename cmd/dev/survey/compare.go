package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
)

func compare(args []string) error {
	if len(args) != 2 {
		usage()
	}

	before, err := readErrors(args[0])
	if err != nil {
		return err
	}
	after, err := readErrors(args[1])
	if err != nil {
		return err
	}

	fmt.Printf("before: %s\nafter:  %s\n", summary(before), summary(after))
	printMissing("removed", before, after)
	printMissing("added", after, before)
	return nil
}

// readErrors reads the errors of a diagnose run, each as "file:line message"
// with the message's first line, counted, since a line can have the same
// error more than once.
func readErrors(path string) (map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	errors := map[string]int{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		var r row
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if r.Severity != int(protocol.DiagnosticSeverityError) {
			continue
		}
		message, _, _ := strings.Cut(r.Message, "\n")
		errors[fmt.Sprintf("%s:%d %s", r.File, r.Line, message)]++
	}

	return errors, scanner.Err()
}

func summary(errors map[string]int) string {
	total := 0
	files := map[string]bool{}
	for key, n := range errors {
		total += n
		file, _, _ := strings.Cut(key, ":")
		files[file] = true
	}
	return fmt.Sprintf("%d errors in %d files", total, len(files))
}

// printMissing prints the errors in one run that the other hasn't.
func printMissing(title string, in, notIn map[string]int) {
	var missing []string
	for key, n := range in {
		for range n - notIn[key] {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)

	fmt.Printf("%s (%d):\n", title, len(missing))
	for _, key := range missing {
		fmt.Println("  " + key)
	}
}
