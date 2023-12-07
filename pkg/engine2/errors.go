package engine2

type (
	ConfigValidationError struct {
		Err error
	}
)

func (e ConfigValidationError) Error() string {
	return e.Err.Error()
}
