package sessionerrs

// Custom is the error type returned by this session service package.
// This custom error is useful for calling funcs to determine which http status code to return to clients on err
type Custom struct {
	// Code corresponds to an http status code (e.g. 401 Unauthorized, or 500 Internal Server Error)
	Code int
	// Err is the actual error thrown
	Err error
}
