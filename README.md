# adr

`boykush` owner のリポジトリに横断して適用する、設計上の決定（ADR）を置くリポジトリ。

## 目的

リポジトリをまたいで繰り返し現れる決定を、リポジトリごとに決め直さず、ここで一度だけ決めて理由とともに残す。

決定は各リポジトリへ複製しない。各リポジトリで作業するエージェントは remote MCP でここを引き、そのリポジトリに書かれていない横断的な決定にも沿って作業する。参照先が一つなので、決め直しもここを変えるだけで全リポジトリに届く。

## 対象

- **置く**: 2つ以上のリポジトリにまたがる決定。対象リポジトリは `boykush` owner のうち fork と archive を除いたもの（github-management の fan-out と同じ範囲）
- **置かない**: 1つのリポジトリに閉じる決定。そのリポジトリに置く
- **置かない**: 決定の適用。設定やファイルの配布・強制は github-management や各リポジトリが担い、ここは何をなぜ決めたかを持つ

## 記録の仕方

[MADR](https://adr.github.io/madr/) 4.0.0 で記録する。1決定1枚、`decisions/NNNN-title-with-dashes.md`。id はファイル名の数字、タイトルは `#` 見出し、status は frontmatter。他の節は MADR のテンプレートに従うが、Confirmation は書かない。守られているかを確かめる基準は、次に述べるルールが持つ。

### ルール

決定とは別に、各リポジトリのセッションが守ることを [ADE](https://github.com/phi42/ad-enforcement-tool) の DSL へ変換したものをルールと呼び、`NNNN-title.rule` として決定と同じ名前で隣に置く。決定はなぜを残し、ルールは何を満たせばよいかだけを言う。効くのは accepted の決定のルールだけ。DSL は ADE のものを使い、独自には作らない。

DSL は散文と違って主語を要求しない——`path "**/CLAUDE.md"` は、それを読んだセッションが作業しているリポジトリについての言明になる。決定の散文はリポジトリを名指さないと語れないので、各リポジトリが何を守るかはルールでしか書けない。

`ade verify` などの実行系は導入しない。ルールを満たすのは読んだセッション自身で、守られているかは各リポジトリの PR のレビューが、差分をルールに照らして確かめる。

ルールにできるのは、決定のうち ADE の DSL で書ける部分だけ。DSL が対象にするのは、構造として現れる決定と、明示的な禁止や要求を述べる決定で、手順やガイドラインの決定は対象外になる。そうした決定はルールを持たず、決定そのものがセッションに伝える。

## 配る道具

`adi`（Architectural Decision Injection）。`adi/` にある。

決定とルールを読んで MCP で配るだけの CLI で、コマンドは `adi mcp` ひとつ。読む側はエージェントなので、人間向けの面は持たない。決定そのものを読むなら `decisions/` を直接開く。

```sh
mise install
mise run build
adi mcp --model decisions            # stdio
adi mcp --model decisions --http 127.0.0.1:8080
```

決定は実行ファイルに埋め込まず `--model` で読む。後でこの道具を別リポジトリへ切り出せる余地を残すため。

### MCP の面

セッションは始めに `list_rules` と `list_decisions` を引き、ルールを満たしつつ、作業に関わる決定を読む。ルールを持たない決定も効くので、ルールだけでは足りない。

| tool | 返すもの |
| --- | --- |
| `list_rules` | accepted な決定の `.rule` すべて |
| `list_decisions` | 全決定の id・title・status |
| `get_decision` | 決定の本文。ルールは含めない |

**書き込みの面は出さない。** 認証も TLS も持たないサーバーを前段越しに公開するので、エージェントは読めるが状態を変えられない形を守る。`.rule` はパースも実行もせず、テキストとして渡すだけ。

ADG の MCP が持つ `get_dsl_reference` と `validate_rule` は、ルールを書く側のためのもの。書くのはこのリポジトリで、読む側のセッションには要らないので持たない。

## 配布

[wiki](https://github.com/boykush/wiki) と同じく remote MCP サーバーとして配る。

- このリポジトリの責務は、`adi` と決定・`.rule` を同梱した image を GHCR へ公開するまで
- Kubernetes の manifest は置かない。infrastructure-as-code の `applications/remote-mcp-server/` に wiki と並べて載せる
- image は `ghcr.io/boykush/adr-mcp-server`。`.github/workflows/adr-mcp-server-image.yml` が main への push で `main` と `<commit 7桁>` の2つの tag を push する
- ENTRYPOINT は `adi mcp --model decisions --http`。呼び出し側が渡すのは listen アドレスだけ（既定 `0.0.0.0:8080`）

## 展望

- [x] 決定を配る MCP サーバーを自作する（[#6](https://github.com/boykush/adr/issues/6)）
- [x] `adi` と決定を同梱した image（`ghcr.io/boykush/adr-mcp-server`）を GHCR へ公開する workflow を置く
- [ ] infrastructure-as-code の remote-mcp-server に載せる
- [ ] レビュー CI から remote MCP を引いて、ルールに照らして差分を見る skill を [ai-plugins](https://github.com/boykush/ai-plugins) から配る
- [ ] dotfiles のグローバル設定から参照させる
