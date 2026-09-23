---
status: accepted
date: 2026-09-23
---

# Choose the manager for external AI plugin dependencies

## Context and Problem Statement

リポジトリの外から来る AI 向けの設定——skill・MCP サーバーの参照・command・agent——に依存する。出どころは2つある。2つ以上のリポジトリで使う物を集めた ai-plugins と、第三者が配る plugin。

Claude Code と Codex を併用している。標準の仕組みはそれぞれ片方にしか効かない。Claude Code は `.claude/settings.json` の `enabledPlugins` で plugin を参照し、Codex は `.codex/config.toml` を project 単位で読む。両方に届けるなら、同じ依存を2箇所へ宣言することになる。

さらに marketplace は、何がどの版で入っているかをエージェントの runtime が持つ。Claude Desktop では `/plugins` で入れ替えてもそのセッションに反映されず、挙動が安定しない。Codex でも同じことが起きる。今どの版が効いているのかを、セッションから確かめられない。

外から来る plugin への依存を、何で宣言するか。

## Decision Drivers

* 効いている版がリポジトリに見え、エージェントの runtime の状態に依存しないこと
* 特定のエージェントに縛られないこと
* 依存の宣言が1箇所で済むこと
* 自前の package と第三者の plugin を同じ宣言で扱えること
* 取得物が固定され、検証して入れられること

## Considered Options

* [microsoft/apm](https://github.com/microsoft/apm)
* Claude Code 標準の plugin marketplace（`.claude/settings.json` の `enabledPlugins`）
* Codex 標準の project config（`.codex/config.toml`）

## Decision Outcome

Chosen option: "microsoft/apm", because 入れた結果がファイルとしてリポジトリに残る。効いている版が `apm.yml` と `apm.lock.yaml` に出るので、エージェントの状態を疑わずに今の版を確かめられ、更新は差分になる。標準の2つはそれぞれ片方のエージェントにしか効かず、両方へ届けるには同じ依存を2箇所へ宣言することになる。marketplace はそのうえ版を runtime が持つので、この確認ができない。

依存は `apm.yml` だけに書き、`targets` には `claude` と `codex` の両方を並べる。生成物は手で編集せず、生成された結果を commit する。片方のエージェントにしか対応しない plugin も、apm から入れる。

### Consequences

* Good, because 宣言1つで Claude と Codex の両方へ届き、自前の package も第三者の plugin も同じ一覧に並ぶ
* Good, because 版の更新が PR の差分に出るので、レビューでき、戻せる
* Bad, because Codex が project の `.codex/` を読むのは trusted なときだけ。untrusted では生成物ごと無視され、リポジトリ側だけでは揃えきれない
* Bad, because apm は 0.x で、生成先の規約が変われば各リポジトリの生成物を作り直すことになる
* Bad, because 依存を変えた PR の差分に生成物が混ざる
* Bad, because 版が見えるようになるだけで、セッションへの反映は開き直すまで待つ
