package githubaction

import (
	"errors"
	"fmt"
	"os"
)

func SetOutput(output, value string) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return errors.New("only output with a github output file supported. See https://github.blog/changelog/2022-10-11-github-actions-deprecating-save-state-and-set-output-commands/ for further details")
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open output file, due %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%v=%v\n", output, value)
	if err != nil {
		return fmt.Errorf("unable to write to output file, due %w", err)
	}

	return nil
}
