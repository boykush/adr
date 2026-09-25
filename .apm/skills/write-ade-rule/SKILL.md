---
name: write-ade-rule
description: decisions/ の .rule ファイルを書く・直すときに使う。決定が選んだ選択肢を、各リポジトリが満たす具体的な形（How）として ADE の rule DSL に書くときや、How が変わってルールだけを直すときの、決定との分担・置き方・書き方・確かめ方を持つ。
---

# ルールを書く

`decisions/NNNN-title.rule` は、隣の決定 `NNNN-title.md` が選んだ選択肢を、各リポジトリのセッションが満たす具体的な形にして、[ADE](https://github.com/phi42/ad-enforcement-tool) の DSL で書いたもの。adi の `list_rules` が、書かれたままを他のリポジトリのセッションへ配る。

## 書く前に読む

- 構文は、skill `ade-rule-dsl` の [references/dsl-reference.md](../ade-rule-dsl/references/dsl-reference.md) を読んでから書く。記憶や似た DSL から推し量らない。go.mod が要求する版の ADE の reference で、`mise run rules` が通す parser と同じ文法を述べている。読む側のセッションも、同じ reference でルールを読む
- 書き方の例は `decisions/0001-choose-the-project-instruction-file-for-coding-agents.rule`

## 決定との分担

- 決定は選択肢の粒度で書かれ、パスや名前やパターンを持たない。それはルールが持つ How。ルールは決定の文面から写すのではなく、選んだ選択肢を何が崩すかから立てる
- なぜそう決めたかは決定が持つ。ルールは何を満たせばよいかだけを言う
- How が変わったら、決定は直さずにルールを直す。コメントに書いた事実が古くなったときも同じ。選んだ選択肢から外れる変更や、決定に書いた理由や帰結が成り立たなくなる変更なら、ルールではなく決定の話で、skill `write-adr` の「直す」に従う
- ルールにできるのは DSL で書ける形だけ。構造として現れるものと、明示的な禁止や要求が対象で、手順やガイドラインは DSL へ押し込まず、実装するリポジトリに任せる。ルールを持たない決定はふつうにある
- 利用側のリポジトリが守るのはルールだけ。決定は理由を知りたいときに読まれるもので、ルールに書かなかったことは利用側を縛らない
- `custom` ブロックは使わない。中身を解釈するのは ADE の plugin で、ここには plugin が無いので、書けば独自の DSL になる

## 書き方

- 決定と同じ名前で隣に置く（`0002-foo.md` なら `0002-foo.rule`）。対になる決定の無い `.rule` があると、adi は決定の読み込みごと失敗する
- 先頭の `adr "<id>" "<title>"` は、決定のファイル名の番号と `#` 見出しに揃える
- パスは、ルールを読んだセッションが作業しているリポジトリの root から見たもの。このリポジトリの配置（`decisions/` や `adi/`）を前提にしない。どこにあっても当てはめるなら `**/` から書く
- そのリポジトリ自身の中身でない場所（worktree、依存パッケージの展開先）は `exclude` で外す
- `#` のコメントもルールと一緒に読む側へ届く。パターンや `exclude` の理由と、違反したときに代わりに使う物のように、ルールを読むだけでは分からないことを書く
- 配られるのは accepted の決定のルールだけ。proposed の決定の隣に書いたルールは、決定が accepted になるまで効かない

## 確かめる

```sh
mise run rules
```

`decisions/` の `.rule` をすべて ADE の parser（`dsl.Validate`）に通す。構文のほか、`adr` 宣言の欠け、ルール名の重なり、未定義の selector、`file` と `code` の assertion の取り違えも見つける。CI も同じ task を回す。

見るのは DSL として読めるかだけで、ルールが満たされているかは見ない。ここは `ade verify` のような実行系を持たず、守られているかは各リポジトリの PR レビューが差分をルールに照らして確かめる。
