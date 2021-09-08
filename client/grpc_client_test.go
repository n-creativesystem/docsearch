package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/n-creativesystem/docsearch/client/models"
	"github.com/n-creativesystem/docsearch/protobuf"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	cmd := exec.Command("test/docsearch", "start")
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	cmd.Env = os.Environ()
	if err := cmd.Start(); !assert.NoError(t, err) {
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	fmt.Println("Sleep")
	time.Sleep(5 * time.Second)
	client, _ := NewGRPCClient("localhost:9000")
	docs := testDocuments()
	reqContent, err := docs.Convert("test")
	if !assert.NoError(t, err) {
		return
	}
	_, err = client.Upload(reqContent)
	assert.NoError(t, err)
	res, err := client.Search(&protobuf.SearchRequest{
		Tenant: "test",
		Query: map[string]*protobuf.Query{
			"match_query": {
				Query: &protobuf.Query_MatchQuery{
					MatchQuery: &protobuf.MatchQuery{
						Match:        "ドリンク飲もうよ",
						Field:        "content",
						AnalyzerName: "kagome",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	var result map[string]interface{}
	err = json.Unmarshal(res.Result, &result)
	assert.NoError(t, err)
	fmt.Printf("%v\n", result)
	fmt.Println(b.String())
	_ = cmd.Process.Kill()
}

func testDocuments() models.Documents {
	testData := []struct {
		ID      string `json:"-"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Author  string `json:"author"`
		Type    string `json:"-"`
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
	var docs = make(models.Documents, 0, len(testData))
	for _, data := range testData {
		doc := models.Document{
			DocumentId: data.ID,
			Fields:     make(map[string]models.FieldMapping),
		}
		doc.Fields["title"] = models.FieldMapping{
			Value:        data.Title,
			AnalyserName: "kagome",
			Type:         models.Keyword,
			StoreValue:   true,
		}
		doc.Fields["content"] = models.FieldMapping{
			Value:        data.Content,
			AnalyserName: "kagome",
			Type:         models.Text,
			StoreValue:   true,
		}
		doc.Fields["author"] = models.FieldMapping{
			Value:        data.Author,
			AnalyserName: "kagome",
			Type:         models.Keyword,
			StoreValue:   true,
		}
		docs = append(docs, doc)
	}
	return docs
}
