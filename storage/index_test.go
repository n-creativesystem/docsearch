package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/search"
	"github.com/blugelabs/bluge/search/aggregations"
	"github.com/ikawaha/blugeplugin/analysis/lang/ja"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type localLogger struct {
	*logrus.Logger
}

func (l *localLogger) Write(p []byte) (int, error) {
	l.Info(p)
	return len(p), nil
}

func TestApp(t *testing.T) {
	log := &localLogger{
		Logger: logrus.New(),
	}
	directory := "../data/docsearch/index"
	index, err := New(directory, log)
	assert.NoError(t, err)
	match := bluge.NewMatchQuery("踊る人形").SetField("content").SetAnalyzer(ja.Analyzer())
	filter := []bluge.Query{
		bluge.NewTermQuery("example1").SetField("type"),
	}
	q := bluge.NewBooleanQuery().AddMust(match).AddMust(filter...)
	request := bluge.NewTopNSearch(10, q).WithStandardAggregations().ExplainScores()
	request.AddAggregation("type", aggregations.NewTermsAggregation(search.Field("type"), 2))
	resDocs, err := index.Search(request)
	log.Info(resDocs)
}

func TestInsertApp(t *testing.T) {
	log := &localLogger{
		Logger: logrus.New(),
	}
	directory := "../data/docsearch/index"
	index, err := New(directory, log)
	assert.NoError(t, err)
	match := bluge.NewMatchQuery("踊っている").SetField("content")
	request := bluge.NewTopNSearch(10, match).WithStandardAggregations().ExplainScores()
	_, _ = index.Search(request)
	docs := []*bluge.Document{}
	doc := bluge.NewDocument("4")
	doc.AddField(bluge.NewTextField("author", "神林長平").WithAnalyzer(ja.Analyzer()).StoreValue())
	doc.AddField(bluge.NewTextField("content", "踊っているのでなければ踊らされているのだろうさ").WithAnalyzer(ja.Analyzer()).StoreValue())
	docs = append(docs, doc)
	index.Update(docs)
	match = bluge.NewMatchQuery("踊っている").SetField("content").SetAnalyzer(ja.Analyzer())
	request = bluge.NewTopNSearch(10, match).WithStandardAggregations().ExplainScores()
	resDocs, err := index.Search(request)
	buf, _ := json.Marshal(resDocs)
	log.Infof("%s", buf)

}

func TestCrud(t *testing.T) {
	log := &localLogger{
		Logger: logrus.New(),
	}
	tmpDir := os.TempDir()
	directory := filepath.Join(tmpDir, "docsearch", "test")
	defer func() {
		os.RemoveAll(directory)
	}()
	index, err := New(directory, log)
	assert.NoError(t, err)
	docs := testDocuments()
	err = index.Update(docs)
	assert.NoError(t, err)
	match := bluge.NewMatchQuery("ゼリー").SetField("content")
	filter := []bluge.Query{
		bluge.NewTermQuery("example1").SetField("type"),
	}
	q := bluge.NewBooleanQuery().AddMust(match).AddMust(filter...)
	request := bluge.NewTopNSearch(10, q).WithStandardAggregations().ExplainScores()
	request.AddAggregation("type", aggregations.NewTermsAggregation(search.Field("type"), 2))
	resDocs, err := index.Search(request)
	assert.NoError(t, err)
	buf, _ := json.Marshal(resDocs)
	log.Infof("%s", buf)
}

func testDocuments() []*bluge.Document {
	testData := []struct {
		ID      string
		Title   string
		Content string
		Author  string
		Type    string
	}{
		{
			ID:    "1",
			Title: "渚橋からグッドモーニング",
			Content: `
			「元田くん、俺もいま逗子だからさ、渚橋の“なぎさカフェ”でコーヒーでも飲まない？　つもる話は、その時に…」 （森山大道・帯文から
			写真家が撮り続けた日常は、いつしか我々の人生と交わっていく。
			元田敬三は路上で出会った人々に声をかけ、まるで恋するように写真を撮る写真家である。スナップとポートレート。多くはモノクロームの男気溢れる世界。そんなまっすぐな写真行為を続け写真家としての確かな評価を集めてきた。結婚し、青年期を過ぎた。人生の伴侶との確かな道のりとともに子供たちは海辺で大きく成長していた。気づけば日常にカメラを向けるようになっていた。日記のように綴られる日付入り写真の集積。しかし当たり前に過ぎていく日々には、本当に大切なことが刻まれていた……。
			舞台は逗子の海辺、東京、各地のストリート。人々との邂逅、子供たちの輝き、そして母の旅立ち……。ページをめくるにつれ写真家の日記は、いつしか我々の人生と交わっていきます。365点にのぼるカラーポジによる写真と日々を綴った文で構成された、見ごたえ読みごたえのある写真集になりました。帯文は森山大道氏が執筆！
			森山氏からは、本作へこんな言葉もいただいています。
			「日記と日録は、しぶとく、したたかな日々の記録＝写真に他ならない。」`,
			Author: "元田　敬三",
			Type:   "example1",
		},
		{
			ID:     "2",
			Title:  "Mr.都市伝説　関暁夫のゾクッとする怪感話",
			Author: "編：BSテレビ東京",
			Content: `
			ゾクッとするけど癖になる!
			Mr.都市伝説 関暁夫が誘う「怪感話」の世界

			BSテレ東で放送中の人気深夜番組が初の書籍化!

			巷で囁かれる都市伝説
			人智を超えた超常現象
			人間の悪意と狂気——

			豪華声優陣による朗読劇24編をノベライズ
			1話5分のゾクッと体験!!!!!!!!!!!

			【コンテンツ】
			01　SNS
			02　いわく付きのマンション
			03　眠れない女優の本当の顔
			04　花嫁になりたかったのに
			05　触れてはいけないアレの秘密
			06　廃病院には近づかない方がいい
			07　死の淵とコーヒーゼリー
			08　それでも運命は変えられない
			09　特殊なアルバイト
			10　運命の出会い、それは……
			11　女の忠告は素直に聞け
			12　住み込みバイト
			13　核戦争の後で
			14　ツーショットダイヤル
			15　死体を溶かす方法
			16　同棲
			17　防犯カメラ
			18　記憶喪失の男
			19　金で買えないものはない
			20　恋するメロンソーダ
			21　廃モーテル
			22　ふらりと入ったバーで
			23　オレオレ詐欺の行方
			24　怖い話にご用心 
			`,
			Type: "example1",
		},
	}
	var docs = make([]*bluge.Document, len(testData))
	for i, data := range testData {
		doc := bluge.NewDocument(data.ID)
		doc.AddField(bluge.NewTextField("title", data.Title).WithAnalyzer(ja.Analyzer()).StoreValue())
		doc.AddField(bluge.NewTextField("content", data.Content).WithAnalyzer(ja.Analyzer()).StoreValue())
		doc.AddField(bluge.NewTextField("author", data.Author).WithAnalyzer(ja.Analyzer()).StoreValue())
		doc.AddField(bluge.NewTextField("type", data.Type).StoreValue())
		docs[i] = doc
	}
	return docs
}
