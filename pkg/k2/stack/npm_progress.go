package stack

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/klothoplatform/klotho/pkg/tui"
)

type NpmProgress struct {
	Progress tui.Progress

	packageCount int
	completed    int
}

// Write parses npm's stdout and stderr to drive setting the progress. It uses loglevel silly output
// and string parsing, so it's not very robust.
func (p *NpmProgress) Write(b []byte) (n int, err error) {
	scan := bufio.NewScanner(bytes.NewReader(b))
	for scan.Scan() {
		line := scan.Text()
		switch {
		case strings.HasPrefix(line, "npm sill tarball no local data"):
			// not in the `npm` cache, need to go fetch it
			p.packageCount++

		case strings.HasPrefix(line, "npm http fetch"):
			// downloading the package
			p.packageCount++
			p.completed++

		case strings.HasPrefix(line, "npm sill ADD"):
			// added the package to node_modules
			p.completed++

			// if the package was in the npm cache, it'll skip straight to the ADD
			// so just add it to the package count
			if p.completed > p.packageCount {
				p.packageCount = p.completed
			}
		}
	}
	if p.packageCount > 0 {
		p.Progress.Update("Installing pulumi packages", p.completed, p.packageCount)
	}
	return len(b), scan.Err()
}
