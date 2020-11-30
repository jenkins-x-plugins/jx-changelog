package helmhelpers

import (
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/pkg/errors"
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
		return "", errors.Wrapf(err, "no Chart.yaml file found in directory '%s'", dir)
	}
	if !exists {
		files, err := filepath.Glob(filepath.Join(dir, "*", "Chart.yaml"))
		if err != nil {
			return "", errors.Wrap(err, "no Chart.yaml file found")
		}
		if len(files) > 0 {
			chartFile = files[0]
		} else {
			files, err = filepath.Glob(filepath.Join(dir, "*", "*", "Chart.yaml"))
			if err != nil {
				return "", errors.Wrap(err, "no Chart.yaml file found")
			}
			if len(files) > 0 {
				for _, file := range files {
					if !strings.HasSuffix(file, "/preview/Chart.yaml") {
						return file, nil
					}
				}
			}
		}
	}
	return chartFile, nil
}
