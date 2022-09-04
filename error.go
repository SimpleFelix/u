package u

import "github.com/SimpleFelix/esg"

type ErrorType = esg.ErrorType

var notWorthLogging byte
var printErrAsInfo byte

// TryConvertToErrorType returns a ErrorType if err is a ErrorType. returns nil if not
func TryConvertToErrorType(err interface{}) ErrorType {
	erro, ok := err.(ErrorType)
	if ok {
		return erro
	}
	return nil
}

func ErrModNoNeedToLog(erro esg.ErrorTypeWriteable) {
	erro.SetExtra(&notWorthLogging)
}

func ErrModPrintAsInfo(erro esg.ErrorTypeWriteable) {
	erro.SetExtra(&printErrAsInfo)
}
