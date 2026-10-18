---
status: accepted
date: 2026-10-18
tags: [javascript, typescript]
---

# Choose the package manager for JavaScript and TypeScript

## Context and Problem Statement

JavaScript と TypeScript のリポジトリは、使うパッケージを npm registry から入れる。npm registry では、正規のパッケージに悪性の版を公開するサプライチェーン攻撃が起きている。パッケージマネージャーの側でできる対策は、install のときに依存のコードを走らせないことと、公開されたばかりの版を入れないこと。

動かすランタイムが Node か Bun かは問わない。使う枠組みがパッケージマネージャーを決めるリポジトリ（Backstage の Yarn など）は、その枠組みを選んだときに決まるので、ここには含めない。

JavaScript と TypeScript のリポジトリは、パッケージを何で入れるか。

## Decision Drivers

* サプライチェーン攻撃への対策が揃っていること
* 設定を書かなくても、既定で対策が効くこと

## Considered Options

* pnpm
* npm
* Yarn (Berry)
* Bun（`bun install`）

## Decision Outcome

Chosen option: "pnpm", because 比べたうちで、依存の install scripts を走らせないことと、公開から日の浅い版を入れないことの両方が、設定を書かなくても効くと文書が述べているのは pnpm だけだった。v11 からは、公開から1日経っていない版を既定で入れない。悪性の版が見つかって取り下げられるまでの間に、入れてしまう見込みが下がる。

ほかの3つも、依存の install scripts は既定で走らせず、日の浅い版を入れない設定を持つ。負けたのは、その設定が既定で効くと文書に書かれていないこと。npm の `min-release-age` は既定では効かない。Yarn の `npmMinimalAgeGate` と Bun の `minimumReleaseAge` は、文書に既定値が書かれていない。

### Consequences

* Bad, because Node に同梱されないので、pnpm を別に入れることになる（[ADR-0002](0002-choose-the-manager-for-tool-dependencies.md)）
* Bad, because npm を前提に書かれた手順や CI の step を、pnpm のものに読み替えることになる
* Bad, because Bun で動かすリポジトリでも、パッケージはそのランタイムの道具ではなく pnpm で入れることになる
* Neutral, because ほかの選択肢との差は、日の浅い版を既定で入れないことにある。ほかが既定でそうなれば選び直す

## More Information

* pnpm の対策: <https://pnpm.io/supply-chain-security>
* pnpm v11.0.0: <https://github.com/pnpm/pnpm/releases/tag/v11.0.0>
* npm の設定: <https://docs.npmjs.com/cli/v12/using-npm/config>
* Yarn の設定: <https://yarnpkg.com/configuration/yarnrc>
* Bun の install: <https://bun.com/docs/pm/cli/install>
