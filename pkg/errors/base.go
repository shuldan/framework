package errors

var newCoreCode = WithPrefix("CORE")

var (
	ErrValidation  = newCoreCode().New("validation failed")
	ErrAuth        = newCoreCode().New("authentication required")
	ErrPermission  = newCoreCode().New("access denied")
	ErrNotFound    = newCoreCode().New("resource not found")
	ErrConflict    = newCoreCode().New("resource conflict")
	ErrBusiness    = newCoreCode().New("business rule violated")
	ErrInternal    = newCoreCode().New("internal error")
	ErrTimeout     = newCoreCode().New("operation timeout")
	ErrUnavailable = newCoreCode().New("service unavailable")
)
