package main

import (
	"os"
	"testing"
)

func TestRunDefaultPackages(t *testing.T) {
	t.Chdir("../..")
	output, err := os.CreateTemp(t.TempDir(), "check-output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()

	if err := run(output); err != nil {
		contents, readErr := os.ReadFile(output.Name())
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Fatalf("default packages must cover the provider API types: %v\n%s", err, contents)
	}
}
