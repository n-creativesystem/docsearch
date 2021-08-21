package analyzer

import "github.com/blugelabs/bluge/analysis"

type config struct {
	charFilter  []analysis.CharFilter
	tokenFilter []analysis.TokenFilter
}

func (c *config) AddCharFilter(filter analysis.CharFilter) {
	c.charFilter = append(c.charFilter, filter)
}

func (c *config) AddTokenFilter(filter analysis.TokenFilter) {
	c.tokenFilter = append(c.tokenFilter, filter)
}

func newConfig() *config {
	return &config{}
}

type baseConfig interface {
	AddCharFilter(filter analysis.CharFilter)
	AddTokenFilter(filter analysis.TokenFilter)
}

type Option func(c baseConfig)

func WithCharFilter(filter analysis.CharFilter) Option {
	return func(c baseConfig) {
		c.AddCharFilter(filter)
	}
}

func WithTokenFilter(filter analysis.TokenFilter) Option {
	return func(c baseConfig) {
		c.AddTokenFilter(filter)
	}
}
