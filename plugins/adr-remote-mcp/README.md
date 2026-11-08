# adr-remote-mcp

この repo の決定を引くための remote MCP サーバーへの参照と、決定が持つルールを読み、照らすための skill を持つ plugin。[Agent Plugins](https://agent-plugins.org/specification) の形で書く。名前の `remote` は、手元で adi を立てる経路と区別するため。

サーバー名は `adr`。tool は `mcp__adr__*` として見える。wiki 側が `scraps`（サーバーを提供するツールの名前）なのに対してこちらを `adr` にしたのは、plugin 名と hostname と揃えたかったから——ツール名で揃えるなら `adi` になる。

## 配る中身

- `mcp.json`: `adr` サーバーへの参照
- skill `ade-rule-dsl`。ルールは [ADE](https://github.com/phi42/ad-enforcement-tool) の DSL で書かれていて、`list_rules` は書かれたままを返す。それを読むための DSL の reference を、サーバーと同じ plugin で届ける
- skill `adr-check` と `adr-review`。変更をルールに照らす手順（[次の節](#adr-check-と-adr-review)）

サーバーの実体は2つの repo に分かれている。

- **image** はこの repo の CI が `ghcr.io/boykush/adr-mcp-server` へ push する。adi に、決定と `.rule` を同梱したもの。したがって引ける内容は **main に push 済みのもの**
- **動く場所と hostname** は [infrastructure-as-code](https://github.com/boykush/infrastructure-as-code) の `applications/remote-mcp-server/adr/`。URL が変わるならここが先に変わり、`mcp.json` が後を追う

読み取り専用で、前段に認証は無い（決定は元から公開）。落ちているときは repo の `decisions/` を直接読む。

### skill の reference

`skills/ade-rule-dsl/references/` の `dsl-reference.md` と `LICENSE` は、ADE の `dsl/dsl-reference.md` と LICENSE（Apache-2.0）を写したもの。手では直さず、`tools/ruledsl` が書く（[README の読み書き](../../README.md#読み書き)）。

### adr-check と adr-review

どちらも変更をルールに照らす手順で、誰が何を照らし、違反をどう扱うかが違う。

| | `adr-check` | `adr-review` |
| --- | --- | --- |
| 照らす側 | 作業したセッション | PR をレビューする側 |
| 照らすもの | 自分の変更 | PR（説明、差分、merge した状態のファイル） |
| 違反があったら | 直す。破らないと作業が成り立たないならユーザーに言う | 指摘として本文に書き、違反あり／なしを判定する |

どちらも単独で使える手順で、互いを前提にしない。同じ変更をローカルと PR のレビューの両方で照らしてもよい。skill が持つのは手順だけで、いつどれを使うかは MCP の instructions が言う。手順まで instructions で語ると、全セッションの handshake に載るうえ、2つの使い方を書き分けきれない。

[boykush/workflows](https://github.com/boykush/workflows) の ai-review は、自分の checkout に入った `adr-review` に従う。判定をどう届けるか（review を出すコマンド）は ai-review の側が持つ。

## 使う

apm で受け取るなら、[ai-plugins](https://github.com/boykush/ai-plugins) がこの plugin から作る package に依存する。apm は Agent Plugins の形を Claude と Codex へは展開しないので、その変換を ai-plugins が持つ。

```yaml
dependencies:
  apm:
    - boykush/ai-plugins/plugins/adr-remote-mcp
```

```bash
apm install
```

Claude は `.mcp.json` と `.claude/skills/`、Codex は `.codex/config.toml` と `.agents/skills/` に入る。どれも project scope なので、リポジトリに commit すればそのリポジトリで開いた全セッションに効く。

`~/.claude/settings.json` の `enabledMcpjsonServers` には `scraps` しか入っていないので、**Claude では初回に `adr` の承認プロンプトが出る**。出したくなければ [dotfiles](https://github.com/boykush/dotfiles) 側に `adr` を足す。
