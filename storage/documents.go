package storage

import (
	"errors"

	"github.com/blugelabs/bluge"
	"github.com/ikawaha/blugeplugin/analysis/lang/ja"
)

type Book struct {
	ID     string
	Author string
	Text   string
}

var (
	docs = []*Book{
		{
			ID:     "1:赤い蝋燭と人魚",
			Author: "小川未明",
			Text:   "人魚は南の方の海にばかり棲んでいるのではありません",
		},
		{
			ID:     "2:吾輩は猫である",
			Author: "夏目漱石",
			Text:   "吾輩は猫である。名前はまだない",
		},
		{
			ID:     "3:狐と踊れ",
			Author: "神林長平",
			Text:   "踊っているのでなければ踊らされているのだろうさ",
		},
		{
			ID:     "4:ダンスダンスダンス",
			Author: "村上春樹",
			Text:   "音楽の鳴っている間はとにかく踊り続けるんだ。おいらの言っていることはわかるかい？",
		},
	}
)

func TestDocuments() []*bluge.Document {
	var ret []*bluge.Document
	for _, v := range docs {
		auth := bluge.NewTextField("author", v.Author).WithAnalyzer(ja.Analyzer())
		body := bluge.NewTextField("body", v.Text).WithAnalyzer(ja.Analyzer())
		doc := bluge.NewDocument(v.ID).AddField(auth).AddField(body)
		ret = append(ret, doc)
	}
	return ret
}

func Exists(id string) (*Book, error) {
	var book Book
	for _, value := range docs {
		if id == value.ID {
			book = *value
			return &book, nil
		}
	}
	return nil, errors.New("No data found")
}
