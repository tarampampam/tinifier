package main

import (
	"github.com/logrusorgru/aurora"
)

type Colors struct {
	au aurora.Aurora
}

var colors Colors

// Enable/Disable colors.
func (c *Colors) enableColors(enabled bool) {
	c.au = aurora.NewAurora(enabled)
}
