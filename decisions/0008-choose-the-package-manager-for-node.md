---
status: accepted
date: 2026-10-18
tags: [node]
---

# Choose the package manager for Node

## Context and Problem Statement

Node のリポジトリは、アプリケーションが import するパッケージを npm registry から入れる。2025 年からは、乗っ取ったメンテナーの権限で悪性の版を公開し、install scripts で広がる攻撃が npm registry で続いている。入れる側にできるのは、公開されたばかりの版や、出どころの怪しい版を、install の時点で入れないことと、入れた版のコードを install のときに走らせないこと。

使う枠組みがパッケージマネージャーを決めるリポジトリ（Backstage の Yarn など）は、その枠組みを選んだときに決まるので、ここには含めない。

Node のリポジトリは、パッケージを何で入れるか。

## Decision Drivers

* サプライチェーン攻撃への対策が揃っていること
* 設定を書かなくても、既定で対策が効くこと

## Considered Options

* pnpm
* npm
* Yarn (Berry)

## Decision Outcome

Chosen option: "pnpm", because 3つのうちで対策が最も揃い、既定で効く範囲も最も広い。v11 からは、依存の install scripts を allowlist の外では走らせず、公開から1日経っていない版を入れないのが既定になった。さらに、公開元の trust が以前の版より下がった版——trusted publisher から公開されていた package の、provenance の無い版など——を入れない設定（`trustPolicy`）を持つ。

npm も v12 で、依存の install scripts を許可した package の外では走らせず、git や tarball の URL から依存を取らないのが既定になった。公開から日の浅い版を入れない設定（`min-release-age`）も v11.10 から持つ。負けたのは、その設定が既定では効かないことと、公開元の trust の格下げで止める仕組みを持たないこと。

Yarn も第三者の package の postinstall を既定で走らせず、公開から日の浅い版を入れない設定（`npmMinimalAgeGate`）を持つ。負けたのは、公開元の trust の格下げで止める仕組みを持たないこと。

### Consequences

* Good, because 新しい版が出ても、既定で1日は入らない。悪性の版が見つかって取り下げられるまでの間に入れる見込みが下がる
* Bad, because Node に同梱されないので、pnpm を別に入れることになる（[ADR-0002](0002-choose-the-manager-for-tool-dependencies.md)）
* Bad, because npm を前提に書かれた手順や CI の step を、pnpm のものに読み替えることになる
* Neutral, because npm との差は、日の浅い版を既定で入れないことと、trust の格下げで止めることの2点。npm がこれを持てば選び直す

## More Information

* pnpm の対策: <https://pnpm.io/supply-chain-security>
* pnpm v11.0.0: <https://github.com/pnpm/pnpm/releases/tag/v11.0.0>
* npm の設定（`allow-scripts`、`allow-git`、`allow-remote`、`min-release-age`）: <https://docs.npmjs.com/cli/v12/using-npm/config>
* Yarn の設定: <https://yarnpkg.com/configuration/yarnrc>
