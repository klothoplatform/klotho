package logging

import (
	"errors"
	"fmt"
	"os"
	"sync"

	tsize "github.com/kopoli/go-terminal-size"
)

var (
	termSize   tsize.Size
	termSizeMu sync.Mutex
)

func TermSize() tsize.Size {
	termSizeMu.Lock()
	defer termSizeMu.Unlock()
	return termSize
}

func init() {
	var err error
	termSizeMu.Lock()
	termSize, err = tsize.GetSize()
	termSizeMu.Unlock()
	switch {
	case errors.Is(err, tsize.ErrNotATerminal):
		termSize = tsize.Size{
			Width:  80,
			Height: 60,
		}

	case err != nil:
		fmt.Fprintf(os.Stderr, "Could not get terminal size: %v\n", err)
		termSize = tsize.Size{
			Width:  80,
			Height: 60,
		}
	}

	if l, err := tsize.NewSizeListener(); err != nil {
		fmt.Fprintf(os.Stderr, "could not create terminal size listener: %v\n", err)
	} else {
		go func(l *tsize.SizeListener) {
			for newSize := range l.Change {
				termSizeMu.Lock()
				termSize = newSize
				termSizeMu.Unlock()
			}
		}(l)
	}
}
