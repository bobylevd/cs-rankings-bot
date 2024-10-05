package cmd

// CommonOpts contains information that is common for all commands.
type CommonOpts struct {
	Version string
}

// Set sets the common options.
func (c *CommonOpts) Set(cc CommonOpts) {
	c.Version = cc.Version
}
