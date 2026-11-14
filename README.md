# adr

`boykush` owner のリポジトリに横断して適用する、設計上の決定（ADR）を置くリポジトリ。

リポジトリをまたいで繰り返し現れる問いを、リポジトリごとに決め直さず、ここで一度だけ決めて理由とともに残す。決定はルールとして各リポジトリの変更に届けられ、PR のレビューや作業中のセッションがそれに照らす。その仕組み——何をもたらし、どの Component と Resource が何を担い、変更がどう届くか——は、他のリポジトリにまたがるので [docs/index.md](docs/index.md)（Backstage では System `adr` の Docs）に書いてある。この README は、このリポジトリそのものを扱う。

## 対象

- **置く**: 2つ以上のリポジトリにまたがる決定。対象リポジトリは `boykush` owner のうち fork と archive を除いたもの（github-management の fan-out と同じ範囲）
- **置かない**: 1つのリポジトリに閉じる決定。そのリポジトリに置く
- **置かない**: 決定の適用。設定やファイルの配布・強制は github-management や各リポジトリが担い、ここは何をなぜ決めたかを持つ

## 置いてあるもの

| パス | 中身 | 書き方 |
| --- | --- | --- |
| `decisions/NNNN-*.md` | 決定。[MADR](https://adr.github.io/madr/) で1決定1枚 | skill `write-adr` |
| `decisions/NNNN-*.rule` | 決定が各リポジトリに求める形を、[ADE](https://github.com/phi42/ad-enforcement-tool) の DSL で書いたルール | skill `write-ade-rule` |
| `adi/` | 決定とルールを MCP で配る CLI | 下の「adi」 |
| `plugins/adr-remote-mcp/` | 配る MCP サーバーへの参照と、ルールに照らす skill を束ねた plugin | [plugin の README](plugins/adr-remote-mcp/README.md) |

決定は「なぜそう選んだか」を、ルールは「何を満たせばよいか」だけを持ち、各リポジトリを縛るのはルールだけ。決定の散文はリポジトリを名指さないと語れないが、DSL は主語を取らず、読んだセッションが作業しているリポジトリについての言明になるので、各リポジトリが何を守るかはルールでしか書けない。DSL は ADE のものを使い、独自には作らない。

### タグ

frontmatter の `tags` に語を並べると、その決定とルールは、同じ語のどれかを名乗ったリポジトリにだけ届く。tags の無い決定とルールは、どのリポジトリにも届く。

ルールのパスでは「Go のリポジトリなら」のような性質を言えないので、その条件を tag としてルールの外で持つ。語は、決定の話題ではなく、リポジトリの性質で立てる。名乗るのはリポジトリの側で、自分について言えることしか名乗れないため。最初の軸は、リポジトリが書かれている言語（`go`）。リポジトリがどの語に当てはまるかは、そのリポジトリの `catalog-info.yaml` にある Component の `metadata.tags` に書く。

`tags` は adr org の [ADG](https://github.com/adr/ad-guidance-tool) が決定の frontmatter に持つ欄に揃えた（MADR に標準の欄は無い）。MADR 本家がカテゴリに使うサブフォルダでは分けない。1つの決定が1つのカテゴリにしか入らず、`ADR-NNNN` の番号もリポジトリの中で一意でなくなるため。

## adi

`adi`（Architectural Decision Injection）は、決定とルールを読んで MCP で配るだけの CLI。読む側はエージェントなので、人間向けの面は持たない。決定そのものを読むなら `decisions/` を直接開く。

```sh
mise install
mise run build
adi mcp --model decisions            # stdio
adi mcp --model decisions --http 127.0.0.1:8080
```

tool・名乗り方・トレースの設定は `adi mcp --help` が、image の作り方は `Dockerfile` と `.github/workflows/adr-mcp-server-image.yml` が持つ。

adr org の ADG の MCP が持つ `get_dsl_reference` と `validate_rule` は持たない。DSL の reference は plugin が skill として MCP サーバーへの参照と一緒に届け、ルールの検査は書く側の `mise run rules` が担う。
