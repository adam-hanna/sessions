package sessions

// setDefaultOptions sets default values for nil fields
// note @adam-hanna: this utility function should be improved. The fields and types of the options struct \
// 			         should not be hardcoded!
func setDefaultOptions(options *Options) {
	emptyOptions := Options{}
	if options.ExpirationDuration == emptyOptions.ExpirationDuration {
		options.ExpirationDuration = DefaultExpirationDuration
	}

	return
}
