package store

// setDefaultOptions sets default values for nil fields
// note @adam-hanna: this utility function should be improved. The fields and types of the options struct \
// 			         should not be hardcoded!
func setDefaultOptions(options *Options) {
	emptyOptions := Options{}
	if options.ConnectionAddress == emptyOptions.ConnectionAddress {
		options.ConnectionAddress = DefaultConnectionAddress
	}
	// note @adam-hanna: what if someone sends in a value of 0? This will set it to default!
	if options.MaxIdleConnections == emptyOptions.MaxIdleConnections {
		options.MaxIdleConnections = DefaultMaxIdleConnections
	}
	// note @adam-hanna: what if someone sends in a value of 0? This will set it to default!
	// if options.MaxActiveConnections == emptyOptions.MaxActiveConnections {
	// 	options.MaxActiveConnections = DefaultMaxActiveConnections
	// }
	if options.IdleTimeoutDuration == emptyOptions.IdleTimeoutDuration {
		options.IdleTimeoutDuration = DefaultIdleTimeoutDuration
	}

	return
}
