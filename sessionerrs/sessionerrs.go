package sessionerrs

// Custom provides a sessions error and also the type of http status code to return
type Custom struct {
	// Code corresponds to http status codes (e.g. 401 Unauthorized)
	Code int
	// Err is the actual error thrown
	Err error
}
