package store

// setDefaultOptions sets default values for nil fields
// note @adam-hanna: this utility function should be improved. The fields and types of the options struct \
// 			         should not be hardcoded!
func setDefaultOptions(opts *Options) {
	emptyOpts := Options{}
	if opts.ConnectionAddress == emptyOpts.ConnectionAddress {
		opts.ConnectionAddress = DefaultConnectionAddress
	}
	if opts.MaxIdleConnections == emptyOpts.MaxIdleConnections {
		opts.MaxIdleConnections = DefaultMaxIdleConnections
	}
	if opts.MaxActiveConnections == emptyOpts.MaxActiveConnections {
		opts.MaxActiveConnections = DefaultMaxActiveConnections
	}
	if opts.IdleTimeoutDuration == emptyOpts.IdleTimeoutDuration {
		opts.IdleTimeoutDuration = DefaultIdleTimeoutDuration
	}
}
