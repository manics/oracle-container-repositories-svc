// Check tags in helm-chart Chart.yaml and values.yaml matches the git tag
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// getGitTag gets the most recent git tag
func getGitTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	gitTag, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(gitTag)), nil
}

// readValues reads a yaml file
func readValues(yamlfile string) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	valuesFile, err := os.ReadFile(yamlfile)
	if err != nil {
		return values, err
	}

	// Unmarshal the values.yaml file
	err = yaml.Unmarshal(valuesFile, &values)
	if err != nil {
		return values, err
	}

	return values, nil
}

func main() {
	errlog := log.New(os.Stderr, "", 0)

	if len(os.Args) != 2 {
		errlog.Println("Usage: check_tags_updated. helm-chart-directory")
		os.Exit(2)
	}
	chartDir := os.Args[1]

	gitTag, err := getGitTag()
	if err != nil {
		errlog.Println(err)
		os.Exit(2)
	}

	chartValues, err := readValues(filepath.Join(chartDir, "Chart.yaml"))
	if err != nil {
		errlog.Println(err)
		os.Exit(2)
	}

	ok := true

	if chartValues["version"] != gitTag {
		errlog.Printf("Chart version '%s' does not match git tag '%s'", chartValues["version"], gitTag)
		ok = false
	}

	if chartValues["appVersion"] != gitTag {
		errlog.Printf("Chart appVersion '%s' does not match git tag '%s'", chartValues["appVersion"], gitTag)
		ok = false
	}

	if !ok {
		os.Exit(1)
	}
}
