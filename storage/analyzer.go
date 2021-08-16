package storage

import "github.com/blugelabs/bluge/analysis"

type AnalyesisFn func() *analysis.Analyzer

var registerAnalyzer = make(map[string]AnalyesisFn)

func Register(name string, analyzer AnalyesisFn) {
	registerAnalyzer[name] = analyzer
}

func GetAnalyzer(name string) *analysis.Analyzer {
	if v, ok := registerAnalyzer[name]; ok {
		return v()
	}
	return nil
}
