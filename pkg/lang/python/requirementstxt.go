package python

import (
	"io"

	klotho_io "github.com/klothoplatform/klotho/pkg/io"
)

type RequirementsTxt struct {
	path string
	// contents is the original pip file's contents. This is shared among clones, so don't modify it!
	contents []byte
	// extras are extra lines that will be appended on write. Each clone has its own.
	extras []string
}

func NewRequirementsTxt(path string, content io.Reader) (*RequirementsTxt, error) {
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, err
	}

	pf := &RequirementsTxt{
		path:     path,
		contents: contentBytes,
	}
	return pf, nil
}

func (pf *RequirementsTxt) AddLine(text string) {
	pf.extras = append(pf.extras, text)
}

func (pf *RequirementsTxt) Clone() klotho_io.File {
	clone := &RequirementsTxt{
		contents: make([]byte, len(pf.contents)),
		extras:   make([]string, len(pf.extras)),
	}
	clone.path = pf.path
	copy(clone.extras, pf.extras)
	copy(clone.contents, pf.contents)
	return clone
}

func (pf *RequirementsTxt) Path() string {
	return pf.path
}

func (pf *RequirementsTxt) WriteTo(out io.Writer) (int64, error) {
	write_count := 0
	b, err := out.Write(pf.contents)
	write_count += b
	if err != nil {
		return int64(write_count), err
	}
	if len(pf.extras) > 0 {
		b, err = out.Write([]byte("\n\n# Added by Klotho:\n"))
		write_count += b
		if err != nil {
			return int64(write_count), err
		}
		for _, extra := range pf.extras {
			b, err = out.Write([]byte(extra + "\n"))
			write_count += b
			if err != nil {
				return int64(write_count), err
			}
		}
	}
	return int64(write_count), nil
}
