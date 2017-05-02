package transport

// setDefaultOptions sets default values for nil fields
// note @adam-hanna: this utility function should be improved. The fields and types of the options struct \
// 			         should not be hardcoded!
func setDefaultOptions(options *Options) {
	emptyOptions := Options{}
	if options.CookieName == emptyOptions.CookieName {
		options.CookieName = DefaultCookieName
	}
	if options.CookiePath == emptyOptions.CookiePath {
		options.CookiePath = DefaultCookiePath
	}
	// note @adam-hanna: how to check for default bool vals? What if someone sends in a value that is false?
	// if options.HTTPOnly == emptyOptions.HTTPOnly {
	// 	options.HTTPOnly = DefaultHTTPOnlyCookie
	// }
	// if options.Secure == emptyOptions.Secure {
	// 	options.Secure = DefaultSecureCookie
	// }

	return
}
