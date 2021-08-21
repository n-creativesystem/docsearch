package analyzer

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNGram(t *testing.T) {
	text := []byte(`福島県三春町の町会議レポート\n先週紹介したように8月16日（木）に「ニコニコ町会議」も参加して福島県三春町の「三春盆踊り」が開催された。お盆休みの最中ということもあってか1万4千250人が来場した。総人口およそ1万7千人の三春町に1万4千人もの人が集まったことになる`)
	kagome := kagomeOriginal()()
	stream := kagome.Analyze(text)
	for _, s := range stream {
		logrus.Info(s.String())
	}
}
