# docsearch

[Golang製の全文検索エンジン](https://github.com/blugelabs/bluge)を利用したアプリケーションです。

日本語アナライザーには[kagome](https://github.com/ikawaha/kagome)を利用しています。

起動時にユーザー辞書を設定するか

APIで公開されているエンドポイントからユーザー辞書を登録して頂くことも可能です。

また、分散合意アルゴリズ([Raft](https://github.com/hashicorp/raft))を利用してクラスターを構成することも可能です。

クラスターを組む際は奇数ノードを指定してください。

## API

[OpenAPI2(swagger)](./protobuf/docsearch.swagger.json)でご確認下さい。

## Architecture

アプリケーションサーバー構成

```
+--client--+  +--------------------------------------------server---------------------------------------------+
|          |  |                                                                                               |
| +------+ |  | +--------------+                                                                              |
| |      | |  | |              |                                                                              |
| | REST -------> gRPC-Gateway -----+                                                                         |
| |      | |  | |              |    |                                                                         |
| +------+ |  | +--------------+    |   +-------------+   +-------------+   +---------+   +-------+           |
| |      | |  |                     |   |             |   |             |   |         |   |       |           |
| | gRPC ---------------------------+---> gRPC-server |---> Raft server |---> storage |---> bluge |           |
| |      | |  |                         |             |   |             |   |         |   |       |           |
| +------+ |  |                         +-------------+   +-------------+   +---------+   +-------+           |
|          |  |                                                                                               |
+----------+  +-----------------------------------------------------------------------------------------------+
```

クラスタリング構成(Raft)

```
                         +-----------+
                         |           |
              +-----------> Follower |
              |          |           |
+--------+    |          +-----------+
|        |    |
| Leader -----+
|        |    |
+--------+    |           +----------+
              |           |          |
              +-----------> Follower |
                          |          |
                          +----------+
```

## Arguments

### `start`

| 引数              | 内容                                                                                |
| :---------------- | :---------------------------------------------------------------------------------- |
| id                | ノードID(default: node1)                                                            |
| raft-address      | raft serverのIPアドレスorホスト名:ポート番号(default: 127.0.0.1:7000)               |
| grpc-address      | grpc serverのIPアドレスorホスト名:ポート番号(default: 127.0.0.1:8000)               |
| http-address      | grpc-gateway(http server)のIPアドレスorホスト名:ポート番号(default: 127.0.0.1:9000) |
| data-directory    | アプリケーション情報を保存するディレクトリパス(default: data/docsearch)             |
| peer-grpc-address | クラスターを組む際の接続先アドレス(bootstrapサーバー)                               |
| certificate-file  | 証明書ファイル                                                                      |
| key-file          | 秘密鍵ファイル                                                                      |
| common-name       | コモンネーム                                                                        |
| log-file          | ログファイル(default: /dev/stdout)                                                  |
| log-level         | ログレベル(default: DEBUG)                                                          |
| log-format        | ログ出力フォーマット(default: json)                                                 |
| user-dictionary   | ユーザー辞書ファイル                                                                |

### Raft(分散合意アルゴリズム)
