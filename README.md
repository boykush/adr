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

[ADG](https://github.com/adr/ad-guidance-tool)（Architectural Decision Guidance）で記録する。ADG は繰り返し現れる決定を扱う道具で、決定を `open` の決定点として起こし、選択肢と判断基準を並べてから確定させる。ファイル構成やコードの依存で検証できる決定は、[ADE](https://github.com/phi42/ad-enforcement-tool) の DSL で `.rule` に落として ADR と並べる。

## セットアップ

使うのは fork の [boykush/ad-guidance-tool](https://github.com/boykush/ad-guidance-tool)。リリースを切っていないので、`mise.toml` の `[bootstrap.repos]` で `.tools/` に clone し、そこから `adg` をビルドする。

```sh
mise run bootstrap
adg --help
```

`mise bootstrap` 全体ではなくタスクを使う。全体だとグローバル設定の repos（`~/dotfiles`）まで対象になるため。fork の版を上げる場所は `mise.toml` の `ref` だけで、書き換えたら `mise run bootstrap` をやり直す。

## 配布

[wiki](https://github.com/boykush/wiki) と同じく remote MCP サーバーとして配る。

- このリポジトリの責務は、`adg` と ADR・`.rule` を同梱した image を GHCR へ公開するまで
- Kubernetes の manifest は置かない。infrastructure-as-code の `applications/remote-mcp-server/` に wiki と並べて載せる

## 展望

- [x] fork で ADG の MCP サーバーを remote（HTTP）で動かせるようにする（[boykush/ad-guidance-tool#1](https://github.com/boykush/ad-guidance-tool/pull/1)）
- [ ] `adg` と決定を同梱した image（`ghcr.io/boykush/adr-mcp-server`）を GHCR へ公開する workflow を置く
- [ ] infrastructure-as-code の remote-mcp-server に載せる
- [ ] dotfiles のグローバル設定から参照させる
