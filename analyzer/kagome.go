package analyzer

import (
	"io"
	"os"

	"github.com/blugelabs/bluge/analysis"
	"github.com/blugelabs/bluge/analysis/token"
	"github.com/ikawaha/blugeplugin/analysis/lang/ja"
	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"golang.org/x/text/unicode/norm"
)

func newKagome(opts ...tokenizer.Option) (*tokenizer.Tokenizer, error) {
	opts = append([]tokenizer.Option{tokenizer.OmitBosEos()}, opts...)
	return tokenizer.New(ipa.DictShrink(), opts...)
}

func kagomeOriginal() func() *analysis.Analyzer {
	t, err := newKagome()
	if err != nil {
		panic(err)
	}
	jt := &ja.JapaneseTokenizer{
		Tokenizer: t,
	}
	kagome := kagomeAnalyzer(jt,
		WithCharFilter(ja.NewUnicodeNormalizeCharFilter(norm.NFKC)),
		WithTokenFilter(token.NewLowerCaseFilter()),
		WithTokenFilter(ja.NewStopWordsFilter()),
		WithTokenFilter(token.NewNgramFilter(2, 3)),
	)
	return func() *analysis.Analyzer {
		return kagome
	}
}

func kagomeAnalyzer(originalTokenizer *ja.JapaneseTokenizer, opts ...interface{}) *analysis.Analyzer {
	config := newConfig()
	for _, opt := range opts {
		switch v := opt.(type) {
		case Option:
			v(config)
		}
	}
	for _, opt := range []ja.TokenizerOption{ja.StopTagsFilter(), ja.BaseFormFilter()} {
		opt(originalTokenizer)
	}
	return &analysis.Analyzer{
		CharFilters:  config.charFilter,
		Tokenizer:    originalTokenizer,
		TokenFilters: config.tokenFilter,
	}
}

func UserDictionary(r io.Reader) (func() *analysis.Analyzer, error) {
	records, err := dict.NewUserDicRecords(r)
	if err != nil {
		return nil, err
	}
	return UserDic(records)
}

func UserDictionaryIsFile(filename string) (func() *analysis.Analyzer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return UserDictionary(file)
}

func UserDic(records dict.UserDictRecords) (func() *analysis.Analyzer, error) {
	d, err := records.NewUserDict()
	if err != nil {
		return nil, err
	}
	t, err := newKagome(tokenizer.UserDict(d))
	if err != nil {
		return nil, err
	}
	originalTokenizer := &ja.JapaneseTokenizer{
		Tokenizer: t,
	}
	analyzer := kagomeAnalyzer(originalTokenizer,
		WithCharFilter(ja.NewUnicodeNormalizeCharFilter(norm.NFKC)),
		WithTokenFilter(token.NewLowerCaseFilter()),
		WithTokenFilter(token.NewNgramFilter(2, 3)),
		WithTokenFilter(ja.NewStopWordsFilter()),
	)
	return func() *analysis.Analyzer {
		return analyzer
	}, nil
}
