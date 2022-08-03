package u

import "github.com/SimpleFelix/esg"

type ErrorType = esg.ErrorType

const NotWorthLogging = "NotWorthLogging"
const PrintErrAsInfo = "PrintErrAsInfo"

// TryConvertToErrorType returns a ErrorType if err is a ErrorType. returns nil if not
func TryConvertToErrorType(err interface{}) ErrorType {
	erro, ok := err.(ErrorType)
	if ok {
		return erro
	}
	return nil
}

func ErrModNoNeedLog(erro esg.ErrorTypeWriteable) {
	erro.SetExtra(NotWorthLogging)
}

func ErrModPrintAsInfo(erro esg.ErrorTypeWriteable) {
	erro.SetExtra(PrintErrAsInfo)
}
