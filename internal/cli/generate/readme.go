//go:build readme

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gh.tarampamp.am/tinifier/v5/internal/cli"
	"gh.tarampamp.am/tinifier/v5/internal/config"
)

func main() {
	const readmePath = "../../README.md"

	_ = os.Setenv(config.DefaultDirPathEnvName, filepath.Join("depends", "on", "your-os"))
	defer func() { _ = os.Unsetenv(config.DefaultDirPathEnvName) }()

	if stat, statErr := os.Stat(readmePath); statErr == nil && stat.Mode().IsRegular() {
		var help = cli.NewApp("tinifier").Help()

		if err := replaceWith(readmePath, help); err != nil {
			panic(err)
		} else {
			fmt.Println("✔ cli docs updated successfully")
		}
	} else if statErr != nil {
		fmt.Println("⚠ readme file not found, cli docs not updated:", statErr.Error())
	}
}

func replaceWith(filePath string, content string) error {
	const start, end = "<!--GENERATED:APP_README-->", "<!--/GENERATED:APP_README-->"

	// read original file content
	original, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	from, to := strings.Index(string(original), start), strings.Index(string(original), end)
	if from == -1 || to == -1 {
		return errors.New("start or end tag not found")
	}

	// write updated content to file
	if err = os.WriteFile(filePath, []byte(strings.Join([]string{
		string(original[:from+len(start)]),
		"## 💻 Command line interface\n",
		"```", content, "```",
		string(original[to:]),
	}, "\n")), 0o664); err != nil {
		return err
	}

	return nil
}
