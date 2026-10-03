---
status: accepted
date: 2026-10-03
tags: [go]
---

# Choose how Go programs read environment variables

## Context and Problem Statement

Go で書くプログラムは、接続先や資格情報といった設定を環境変数から受け取る。環境変数は文字列でしか渡らないので、既定値、必須かどうか、型への変換は読む側が持つ。依存するライブラリが自分で読む環境変数は、ここには含めない。

Go のプログラムは、環境変数を何で読むか。

## Decision Drivers

* 読む環境変数の名前・型・既定値・必須かどうかが、1つの宣言に集まること
* 欠けた変数や型に合わない値が、起動時にまとめて分かること
* 環境変数を読むことだけを足し、他の機能や依存を連れてこないこと

## Considered Options

* [caarlos0/env](https://github.com/caarlos0/env)
* 標準ライブラリの `os.Getenv` / `os.LookupEnv` を直接呼ぶ
* [spf13/viper](https://github.com/spf13/viper)

## Decision Outcome

Chosen option: "caarlos0/env", because 環境変数ごとの名前・型・既定値・必須かどうかを、struct の field とその tag に書く。1回の parse で、欠けた必須の変数も型に合わない値も、最初の1つで止まらずに集めて返す。扱うのは環境変数だけで、依存する module を持たない。

標準ライブラリを直接呼べば何も足さずに済むが、既定値・必須の判定・型への変換を呼び出しごとに書くことになり、読む変数は呼び出し箇所に散る。不備が分かるのも、その呼び出しに届いたとき。

viper は環境変数を key に結び付けて読む。名前は結び付けるとき、型は値を取り出すとき、既定値は別の呼び出しで決まるので、1つの宣言にならない。必須の指定は無い。設定ファイル・flag・remote の設定元まで扱い、そのための依存を連れてくる。

### Consequences

* Good, because テストでは環境の代わりに map を渡せて、プロセスの環境を書き換えずに済む
* Bad, because 環境変数を1つ読むだけの箇所でも、struct と依存が1つずつ増える
* Bad, because flag と環境変数の両方で受ける値は、どちらを優先するかを自分で書くことになる

## More Information

* 集めたエラーを返す型: <https://pkg.go.dev/github.com/caarlos0/env/v11#AggregateError>
* viper の環境変数の扱い: <https://github.com/spf13/viper#working-with-environment-variables>
