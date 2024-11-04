package helmhelpers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
)

const (
	// ChartFileName file name for a chart
	ChartFileName = "Chart.yaml"
)

// FindChart find a chart in the current working directory, if no chart file is found an error is returned
func FindChart(dir string) (string, error) {
	chartFile := filepath.Join(dir, ChartFileName)
	exists, err := files.FileExists(chartFile)
	if err != nil {
		return "", fmt.Errorf("no Chart.yaml file found in directory '%s': %w", dir, err)
	}
	if !exists {
		fs, err := filepath.Glob(filepath.Join(dir, "*", "Chart.yaml"))
		if err != nil {
			return "", fmt.Errorf("no Chart.yaml file found: %w", err)
		}
		if len(fs) > 0 {
			chartFile = fs[0]
		} else {
			fs, err = filepath.Glob(filepath.Join(dir, "*", "*", "Chart.yaml"))
			if err != nil {
				return "", fmt.Errorf("no Chart.yaml file found: %w", err)
			}
			if len(fs) > 0 {
				for _, file := range fs {
					if !strings.HasSuffix(file, "/preview/Chart.yaml") {
						return file, nil
					}
				}
			}
		}
	}
	return chartFile, nil
}
