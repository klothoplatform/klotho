package iac2

type (
	DockerLambda struct {
		ExecUnitName    string
		CloudwatchGroup CloudwatchLogGroup
	}

	CloudwatchLogGroup struct {
		// fields...
	}
)

func (cwg CloudwatchLogGroup) VariableName() string {
	return "cloudwatchLogGroup"
}

func (dl DockerLambda) VariableName() string {
	return "lambda_" + nonIdentifierChars.ReplaceAllString(dl.ExecUnitName, "_")
}
