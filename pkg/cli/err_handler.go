package cli

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type ErrorHandler struct {
	InternalDebug bool
	Verbose       bool
	PostPrintHook func()
}

func (h ErrorHandler) PrintErr(err error) {
	h.printErr(err, 0)
	if h.PostPrintHook != nil {
		h.PostPrintHook()
	}
}

func (h ErrorHandler) printErr(err error, num int) (nextNum int) {
	log := zap.L()

	errFmt := "%v"
	if h.InternalDebug {
		errFmt = "%+v"
	} else if h.Verbose {
		errFmt = "%#v"
	}

	merr, ok := err.(multierr.Error)
	if ok {
		switch len(merr) {
		case 0:
			return

		case 1:
			err = merr[0]

		default:
			log.Sugar().Errorf("%d errors:", len(merr))
			for _, err := range merr {
				num = h.printErr(err, num+1)
			}
			return num
		}
	}

	msg := ""
	suberr := err
	for suberr != nil {
		suberr = errors.Unwrap(suberr)
		if suberr == nil {
			break
		}

		switch suberr := suberr.(type) {
		case *klotho_errors.WrappedError:
			if msg == "" {
				msg = suberr.Message
			} else {
				msg += ": " + suberr.Message
			}
		case *types.CompileError:
			if msg == "" {
				log.
					With(logging.FileField(suberr.File), logging.AnnotationField(suberr.Annotation)).
					Sugar().
					Errorf("[err %d] "+errFmt, num+1, suberr.Cause)
			} else {
				log.
					With(logging.FileField(suberr.File), logging.AnnotationField(suberr.Annotation)).
					Error(
						fmt.Sprintf("[err %d] "+errFmt, num+1, msg),
						logging.PostLogMessageField(fmt.Sprintf("-> Caused by: "+errFmt, suberr.Cause)),
					)
			}
			return num
		}
	}

	log.Sugar().Errorf("[err %d] "+errFmt, num, err)

	return num
}
