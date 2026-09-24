# adr

`boykush` owner のリポジトリに横断して適用する、設計上の決定（ADR）を置くリポジトリ。

## 目的

リポジトリをまたいで繰り返し現れる決定を、リポジトリごとに決め直さず、ここで一度だけ決めて理由とともに残す。

決定は各リポジトリへ複製しない。各リポジトリで作業するエージェントは、作業を終えて push する前に remote MCP でここを引き、そのリポジトリに書かれていない横断的な決定にも変更が沿っているかを確かめる。参照先が一つなので、決め直しもここを変えるだけで全リポジトリに届く。

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

#### 読み書き

DSL の reference は、ルールを読む側にも書く側にも要る。それを持つ skill `ade-rule-dsl` は、[ai-plugins](https://github.com/boykush/ai-plugins) の `adr-remote-mcp` package が MCP サーバーへの参照と一緒に配る。ルールを受け取るリポジトリには、読むための文法も同じ依存で届く。この repo もその package に依存して同じ skill を受け取る（MCP サーバー `adr` の参照も一緒に入る）。

書き方は skill `write-ade-rule` が持つ。決定との対応やパスの基準といった、この repo だけの取り決めなので ai-plugins には置かず、`.apm/skills/` から `apm install` が `.claude/skills/` と `.agents/skills/` へ配る。文法は reference を分けず、`ade-rule-dsl` のものを読む。書いたら `mise run rules` で ADE の parser（`dsl.Validate`）に通す。中身は `tools/ruledsl` で、CI も同じ task を回す。確かめるのは DSL として読めることだけで、ルールが満たされているかは見ない。

reference は ADE の `dsl/dsl-reference.md` を、Apache-2.0 の LICENSE と一緒に写したもの。上流を指すだけにしないのは、Codex の sandbox のようにネットワークに出られないセッションでも読めるようにするため。版は go.mod が要求する ADE、つまり `mise run rules` が通す parser の版に揃え、この repo に配られた reference がその版のものかを `go test` が確かめる。ADE を上げたら ai-plugins 側を書き直し、その commit へ `apm.yml` の固定を上げる。

```sh
go run ./tools/ruledsl vendor <ai-plugins>/plugins/adr-remote-mcp/.apm/skills/ade-rule-dsl/references
```

### タグ

frontmatter の `tags` に語を並べると、その決定は、同じ語のどれかを宣言したリポジトリにだけ一覧される。tags の無い決定は、どのリポジトリにも一覧される。宣言のしかたは [MCP の面](#mcp-の面) に書く。

MADR に標準の欄は無いので、adr org の [ADG](https://github.com/adr/ad-guidance-tool) が決定の frontmatter に持つ `tags` に揃えた。MADR 本家がカテゴリに使うサブフォルダでは分けない。1つの決定が1つのカテゴリにしか入らず、`ADR-NNNN` の番号もリポジトリの中で一意でなくなるため。

何を軸に語を立てるかはまだ決めていないので、今はどの決定にも付けない。

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

セッションは作業を終えて push する前に `list_rules` と `list_decisions` を引き、変更がルールを満たすかを確かめ、変更に関わる決定を読む。始めに引かないのは、作業に関わらないルールや決定までセッションに載ってノイズになるため。ルールを持たない決定も効くので、ルールだけでは足りない。

| tool | 返すもの |
| --- | --- |
| `list_rules` | accepted な決定の `.rule` すべて |
| `list_decisions` | 決定の id・title・status・tags。利用側が tag を宣言していれば、そのどれかを持つ決定と tags の無い決定に絞る |
| `get_decision` | 決定の本文と tags。ルールは含めない |

利用側は、自分に当てはまる [tag](#タグ) をカンマ区切りで宣言する。HTTP ではリクエストの `Adi-Tags` header に、stdio では環境変数 `ADI_TAGS` に書く。HTTP のサーバーは1つで全リポジトリに答えるので、宣言はリクエストごとに運び、サーバー自身の環境変数は見ない。

絞るのは `list_decisions` だけ。ルールはパスで自分の対象を限っているので `list_rules` は絞らず、`get_decision` は id で指された決定をそのまま返す。

apm で `adr` サーバーを受け取るリポジトリが宣言するときは、自分の `apm.yml` の `dependencies.mcp` に `adr` を `headers` 付きで宣言し直す。apm は同じ名前のサーバーを root の宣言を優先して1つにするので、package の宣言が置き換わる。値は `${VAR}` にせず直接書く。apm は `${VAR}` を install 時に解決して生成物へ焼き込むので、宣言の出どころが install した環境になる。

**書き込みの面は出さない。** 認証も TLS も持たないサーバーを前段越しに公開するので、エージェントは読めるが状態を変えられない形を守る。`.rule` はパースも実行もせず、テキストとして渡すだけ。

ADG の MCP が持つ `get_dsl_reference` と `validate_rule` は持たない。reference は読む側にも要るが、MCP サーバーへの参照と同じ package が skill として届けるので、MCP の面には足さない。handshake の instructions がその skill を指す。`validate_rule` の役は、ルールを書くこの repo の `mise run rules` が持つ（[読み書き](#読み書き)）。

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
- [ ] [tag](#タグ) の軸と語彙を決め、決定に付けて、各リポジトリが宣言する
