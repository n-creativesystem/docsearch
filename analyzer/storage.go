package analyzer

import (
	"strings"
	"sync"

	"github.com/blugelabs/bluge/analysis"
	"github.com/blugelabs/bluge/analysis/analyzer"
)

var _analyzer = &registerAnalyzer{
	analyzer: make(map[string]AnalyesisFn),
}

var standardAnalyzer = analyzer.NewStandardAnalyzer()

func init() {
	Register("", func() *analysis.Analyzer {
		return standardAnalyzer
	})
	Register("kagome", kagomeOriginal())
}

type AnalyesisFn func() *analysis.Analyzer

type registerAnalyzer struct {
	mu       sync.RWMutex
	analyzer map[string]AnalyesisFn
}

func (a *registerAnalyzer) register(name string, analyzer AnalyesisFn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.analyzer[strings.ToLower(name)] = analyzer
}

func (a *registerAnalyzer) getAnalyzer(name string) *analysis.Analyzer {
	a.mu.Lock()
	defer a.mu.Unlock()
	if v, ok := a.analyzer[strings.ToLower(name)]; ok {
		return v()
	}
	return nil
}

func (a *registerAnalyzer) remove(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.analyzer, name)
}

func Register(name string, analyzerFn AnalyesisFn) {
	_analyzer.register(name, analyzerFn)
}

func GetAnalyzer(name string) *analysis.Analyzer {
	return _analyzer.getAnalyzer(name)
}

func Remove(name string) {
	_analyzer.remove(name)
}
