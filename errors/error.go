package errors

import "fmt"

func New(f string, args ...interface{}) error {
	return fmt.Errorf(f, args...)
}

func NodeInfo(err error) error {
	return formatErr("ノード情報の取得に失敗しました", err)
}

func ClusterInfo(err error) error {
	return formatErr("クラスター情報の取得に失敗しました", err)
}

func RequestForward(target string, err error) error {
	err = formatErr("リクエストの転送に失敗しました", err)
	return fmt.Errorf("%v: [target: %s]", err, target)
}

func NodeLeave(err error, value interface{}) error {
	err = formatErr("クラスターからノードを離すことが出来ませんでした", err)
	return fmt.Errorf("%v: [value: %v]", err, value)
}

func JoinNode(err error, value interface{}) error {
	err = formatErr("ノードがクラスターに参加できませんでした", err)
	return fmt.Errorf("%v: [value: %v]", err, value)
}

func RaftConfigration(err error) error {
	return formatErr("Raft configurationが取得できませんでした", err)
}

func AddVoter(err error) error {
	return formatErr("Voterの追加に失敗しました", err)
}

func CommandUnmashal(err error, id string, metadata interface{}) error {
	return fmt.Errorf("%v: [id:%s],[metadata:%v]", formatErr("コマンドデータの変換(Unmarshal)に失敗しました", err), id, metadata)
}

func formatErr(message string, err error) error {
	return fmt.Errorf("%s: %s", message, err.Error())
}

var (
	NodeAlreadyExists = New("既にノードが存在しています")
	TimeOut           = New("タイムアウト")
	NotFoundLeader    = New("リーダーノードが存在しません")
	NotFound          = New("存在しません")
	NodeDoesNotExist  = New("ノードが存在しません")
	Nil               = New("data is nil")
)
