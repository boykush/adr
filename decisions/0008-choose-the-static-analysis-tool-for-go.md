---
status: accepted
date: 2026-10-10
tags: [go]
---

# Choose the static analysis tool for Go

## Context and Problem Statement

Go のツールチェーンには `go vet` が付いてくる。`go vet` は誤検知をほとんど出さないものに検査を絞っていて、使われないコードや無視されたエラー、非推奨の API の呼び出しのような、それより踏み込んだ解析は持たない。

Go のリポジトリは、静的解析を何で行うか。

## Decision Drivers

* 標準の `go vet` の検査を使えること
* その上で、`go vet` より踏み込んだ解析を使えること

## Considered Options

* [golangci-lint](https://golangci-lint.run/)
* `go vet` のみ

## Decision Outcome

Chosen option: "golangci-lint", because `go vet` の検査を `govet` という linter として含んだまま、staticcheck や errcheck のような踏み込んだ解析を同じ実行に並べられる。

`go vet` のみでは、標準の検査は使えるが、それより踏み込んだ解析が無い。

### Consequences

* Good, because 足す解析を、設定で1つずつ有効にも無効にもできる
* Bad, because 解析器の版は golangci-lint の release に従い、上流の解析器を個別には上げられない
* Bad, because golangci-lint を上げると、コードを変えていなくても新しい指摘が出ることがある
* Bad, because golangci-lint は、自身を build した Go より新しい Go のコードを解析できないので、Go を上げるときに待たされることがある

## More Information

* golangci-lint が含む linter: <https://golangci-lint.run/docs/linters/>
* `go vet` が検査を絞る基準: <https://github.com/golang/go/blob/master/src/cmd/vet/README>
