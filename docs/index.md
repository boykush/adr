# 決定が変更に届くまで

## この System がもたらすもの

boykush 配下の repo に横断して効く技術的な意思決定を一度だけ決め、それが求めるルールを、決定が当てはまる repo の変更に届ける。repo が増えても、一つひとつで同じ問いを決め直さずに済み、全体が同じ答えに揃う——統制が、人の目ではなく仕組みで効く。

- **一度決めれば、全 repo が揃う。** 「ツールの版はどこで宣言するか」「GITHUB_TOKEN で足りない workflow は何で認証するか」のような、repo ごとに答えが割れがちな問いを [decisions/](https://github.com/boykush/adr/tree/main/decisions) で一度だけ決める。決め直しも、ここを変えるだけで全 repo に及ぶ。
- **当てはまる repo にだけ効く。** 「Go の静的解析に何を使うか」のような決定は、Go の repo にだけ届く。全 repo に当てる決定と、性質の合う一部の repo に当てる決定を、同じ置き場で持てる。
- **決めたことが、実際の変更で守られる。** 決定は読まれることを当てにしない。PR のレビューと作業中のエージェントが、変更をルールに照らし、違反を入る前に指摘する。決めた側が各 repo を見回らなくても、外れた変更はそこで見つかる。

このページの残りは、それがどんな役割の組み合わせで動き、変更がどう届くかを書く。決定の書き方や MCP の面の細部は [README](https://github.com/boykush/adr#readme) が持つ。

## 全体像

関わる entity と関係は、[System `adr`](https://backstage.boykush.com/catalog/default/system/adr) と [API `adr-mcp`](https://backstage.boykush.com/catalog/default/api/adr-mcp) のページの Graph が catalog から描く。どの entity がどの役割を担うかも、それぞれの entity の catalog が持つ。

仕組みは、決める・配る・照らす・名乗るの4つの役割で動く。System `adr` に入っているのは決める側だけで、残りは周りの repo と基盤が担う。

- **決める**　決定と、それが各 repo に求めるルールを [adr](https://backstage.boykush.com/catalog/default/component/adr) に置く。
- **配る**　ルールは [adr-mcp](https://backstage.boykush.com/catalog/default/api/adr-mcp)（[infrastructure-as-code](https://backstage.boykush.com/catalog/default/component/infrastructure-as-code) が cluster で動かす）が、照らし方の skill は plugin の [adr-remote-mcp](https://backstage.boykush.com/catalog/default/airesource/adr-remote-mcp) が届ける。merge してから効くまでに、ルールは image の入れ替わりを、skill は各 repo の版の更新を待つ。
- **照らす**　ai-review（[workflows](https://backstage.boykush.com/catalog/default/component/workflows)）を入れた repo では PR のレビューが、それ以外ではエージェントが push の前に、変更をルールに照らす。cluster が止まる夜間は adr-mcp に繋がらず、ai-review は失敗する。
- **名乗る**　各 repo は catalog の自分の Component の tags で性質（書かれている言語など）を名乗り、tag の付いた決定はそれを名乗った repo にだけ届く。
