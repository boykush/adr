---
status: accepted
date: 2026-09-25
---

# Choose the home for repository-local AI plugins

## Context and Problem Statement

外から来る skill・MCP サーバーの参照・agent は apm で宣言している。一方、そのリポジトリでしか使わない物もある。そのリポジトリだけの取り決めを持つ skill のように、2つ以上のリポジトリで使わないので外へ出す理由が無い物。

置き場はエージェントごとに違う。Claude Code は `.claude/skills/`、Codex は `.agents/skills/` を読む。手で置くなら、同じ中身がエージェントの数だけ要る。

さらに、apm が外から来る物しか持たないと、今そのリポジトリに何が効いているかが2つの仕組みに分かれる。`apm install` が配った物は `apm.lock.yaml` に配り先まで残り、`apm audit` が手書きの変更を drift として見つける。手で置いた物はどちらにも載らない。

そのリポジトリだけで使う skill 等を、どこに置き、どうやって各エージェントへ届けるか。

## Decision Drivers

* 同じ中身を、エージェントごとの置き場へ写さないこと
* 特定のエージェントに縛られないこと
* 外から来る依存と自前の物を、同じ宣言と同じ検査で扱えること
* 効いている物が手で書き換えられていないと確かめられること

## Considered Options

* `.apm/` に置き、`apm install` に各エージェントの置き場へ展開させる
* エージェントごとの置き場（`.claude/skills/` と `.agents/skills/`）へ直接置く
* 片方を実体にし、もう片方から symlink で繋ぐ
* 自前の物も ai-plugins の package にして、外部依存として入れ直す

## Decision Outcome

Chosen option: "`.apm/` に置き、`apm install` に各エージェントの置き場へ展開させる", because apm はリポジトリ自身を local の package として扱い、`.apm/` の中身を `targets` の各エージェントの置き場へ配って、配り先を `apm.lock.yaml` に記録する。外から来る package と自前の物が同じ宣言に並び、`apm audit` はどちらの手書きの変更も drift として見つける。直接置けば同じ中身がエージェントの数だけ増え、どれを直すのかを別に決めることになる。symlink は git に入れても環境によって実体にならず、エージェントが辿る保証も無い。ai-plugins へ出すのは、1つのリポジトリに閉じる物を2リポジトリに分け、直すたびに固定した commit を上げ直すことになる。

### Consequences

* Good, because 自前の物も外から来る物も `apm install` ひとつで揃い、何がどこへ配られたかが `apm.lock.yaml` に出る
* Good, because 生成物への手書きの変更を `apm audit` が drift として見つけ、`apm install` が書き戻す
* Bad, because 直した物が効くのは `apm install` の後で、`.apm/` を直しただけではセッションに届かない
* Bad, because 自前の物が、ソースとエージェントごとの写しとして、PR の差分に何度も出る

## More Information

* apm を選んだ理由は [ADR-0003 Choose the manager for external AI plugin dependencies](0003-choose-the-manager-for-external-ai-plugin-dependencies.md)。ここはその宣言に自前の物を載せるだけで、manager は選び直していない
