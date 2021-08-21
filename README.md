# docsearch

[Golang製の全文検索エンジン](https://github.com/blugelabs/bluge)を利用したアプリケーションです。

日本語アナライザーには[kagome](https://github.com/ikawaha/kagome)を利用しています。

起動時にユーザー辞書を設定するか

APIで公開されているエンドポイントからユーザー辞書を登録して頂くことも可能です。

## Architecture

- client(http) -> grpc_gateway -> grpc_server -> grpc_service -> raft_server -> raft -> storage -> bluge
- client(grpc) -> grpc_server -> grpc_service -> raft_server -> raft -> storage -> bluge
