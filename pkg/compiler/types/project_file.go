package types

import (
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/zap"
)

type projectFileMatch struct {
	execUnitPath string
	path         string
	exactMatch   bool
}

// CheckForProjectFile will find an existing inputFile, corresponding to the filename param,that best matches the path where the execution unit annotation
// lives and returns the path to the corresponding project file
func CheckForProjectFile(inputFiles *InputFiles, unit *ExecutionUnit, filename string) string {
	log := zap.L().With(zap.String("unit", unit.Name)).Sugar()
	currMatch := projectFileMatch{}

	unitDeclarers := unit.GetDeclaringFiles()

	// Sorting the unit's declaring files by path makes project file selection more deterministic
	sort.Slice(unitDeclarers, func(i, j int) bool { return unitDeclarers[i].Path() > unitDeclarers[j].Path() })

	var projectFilePaths []string
	for _, inputFile := range inputFiles.Files() {
		if filename == filepath.Base(inputFile.Path()) {
			projectFilePaths = append(projectFilePaths, inputFile.Path())
		}
	}

	for _, declaringFile := range unitDeclarers {
		pMatch := findBestMatch(projectFilePaths, declaringFile.Path())
		if pMatch == (projectFileMatch{}) {
			continue
		}
		if currMatch == (projectFileMatch{}) {
			currMatch = pMatch
		}
		if currMatch.exactMatch {
			if pMatch.exactMatch && currMatch.path != pMatch.path {
				log.Warnf(`Found multiple project files. Using "%s" instead of "%s."`, currMatch.path, pMatch.path)
			}
		} else if pMatch.exactMatch {
			currMatch = pMatch
		} else {
			if currMatch.path == filename {
				currMatch = pMatch
			}
			if pMatch.path != currMatch.path && currMatch.path != filename {
				log.Warnf(`Found multiple project files. Using "%s" instead of "%s."`, currMatch.path, pMatch.path)
			}
		}
	}

	return currMatch.path
}

func findBestMatch(paths []string, path string) projectFileMatch {
	basepath := path
	for range strings.Split(path, string(filepath.Separator)) {
		basepath = filepath.Dir(basepath)
		for _, p := range paths {
			match, err := filepath.Match(basepath, filepath.Dir(p))
			if err != nil {
				continue
			}
			if match {
				return projectFileMatch{
					path:         p,
					exactMatch:   basepath == filepath.Dir(path),
					execUnitPath: path,
				}
			}
		}
	}
	return projectFileMatch{}
}
