---
status: accepted
date: 2026-09-21
---

# Choose the manager for tool dependencies

## Context and Problem Statement

リポジトリの開発には、言語ランタイムや CLI といった道具が要る。アプリケーションが import するライブラリは言語のパッケージマネージャーが扱うので、ここには含めない。

リポジトリが使う道具を、何で宣言するか。

## Decision Drivers

* どこで入れても、同じ宣言から同じ版が入ること
* 言語ランタイムと CLI を1箇所で宣言できること
* 取得物を検証して入れられること

## Considered Options

* mise
* aqua
* Nix（flakes や devbox）

## Decision Outcome

Chosen option: "mise", because 言語ランタイムも CLI も `mise.toml` の `[tools]` に宣言できる。mise は aqua registry の定義を使って CLI を入れられ、定義があれば署名や provenance まで検証する。aqua だけでは Rust や Python のランタイムを別に入れることになる。Nix は両方を宣言できるが、単一のバイナリで済む mise と違い、使う環境ごとに /nix の store を用意することになる。

道具は `mise.toml` の `[tools]` に宣言する。mise は `.tool-versions` も読むが、宣言の場所は1つにする。

### Consequences

* Good, because 道具とその版が `mise.toml` に集まり、どこでも同じ宣言から入る
