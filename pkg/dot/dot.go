package dot

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/pprof/third_party/svgpan"
	"go.uber.org/zap"
)

// THe following adds SVG pan to the SVG output from DOT, taken from
// https://github.com/google/pprof/blob/main/internal/driver/svg.go

var (
	viewBox  = regexp.MustCompile(`<svg\s*width="[^"]+"\s*height="[^"]+"\s*viewBox="[^"]+"`)
	graphID  = regexp.MustCompile(`<g id="graph\d"`)
	svgClose = regexp.MustCompile(`</svg>`)
)

// SvgPan enhances the SVG output from DOT to provide better
// panning inside a web browser. It uses the svgpan library, which is
// embedded into the svgpan.JSSource variable.
func SvgPan(svg string) string {
	// Work around for dot bug which misses quoting some ampersands,
	// resulting on unparsable SVG.
	svg = strings.Replace(svg, "&;", "&amp;;", -1)

	// Dot's SVG output is
	//
	//    <svg width="___" height="___"
	//     viewBox="___" xmlns=...>
	//    <g id="graph0" transform="...">
	//    ...
	//    </g>
	//    </svg>
	//
	// Change it to
	//
	//    <svg width="100%" height="100%"
	//     xmlns=...>

	//    <script type="text/ecmascript"><![CDATA[` ..$(svgpan.JSSource)... `]]></script>`
	//    <g id="viewport" transform="translate(0,0)">
	//    <g id="graph0" transform="...">
	//    ...
	//    </g>
	//    </g>
	//    </svg>

	if loc := viewBox.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`<svg width="100%" height="100%"` +
			svg[loc[1]:]
	}

	if loc := graphID.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`<script type="text/ecmascript"><![CDATA[` + svgpan.JSSource + `]]></script>` +
			`<g id="viewport" transform="scale(0.5,0.5) translate(0,0)">` +
			svg[loc[0]:]
	}

	if loc := svgClose.FindStringIndex(svg); loc != nil {
		svg = svg[:loc[0]] +
			`</g>` +
			svg[loc[0]:]
	}

	return svg
}

func Execute(input io.Reader, output io.Writer) error {
	errBuff := new(bytes.Buffer)
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = errBuff
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not run 'dot': %w: %s", err, errBuff.String())
	}
	return nil
}

func ExecPan(input io.Reader) (string, error) {
	out := new(bytes.Buffer)
	err := Execute(input, out)
	if err != nil {
		return "", err
	}
	zap.S().Named("dot").Debugf("dot output %d bytes", out.Len())
	return SvgPan(out.String()), nil
}
