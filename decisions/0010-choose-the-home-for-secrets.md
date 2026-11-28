---
status: accepted
date: 2026-11-28
---

# Choose the home for secrets

## Context and Problem Statement

boykush 配下で使う秘密——外部サービスの API token や、アプリが使う資格——は、CI の workflow、クラスタで動くアプリ、手元での作業のそれぞれから読まれる。**1つの秘密を複数の場所が読む**ことは珍しくない。CI と手元の両方が同じ API token を使う、といった形で、リポジトリをまたぐかどうかに関わらず起きる。

owner は User アカウントなので organization secret が無く、GitHub の secret はリポジトリごとに置くしかない。repository secret は書き込み専用で、手元からは読めない。使う場所ごとの仕組みに置くと、同じ秘密の写しが場所の数だけでき、入れ替えはその数だけ要り、何がどこにあっていつ切れるかを一覧する手段も無い。

[ADR-0004](0004-choose-the-home-for-secrets-shared-across-repositories.md) は、この問いを「複数のリポジトリで共有する秘密」に限って AWS を選んだ。しかし複数の場所から読まれることは共有に限らず、共有かどうかで置き場を分ける理由は薄い。範囲を秘密すべてに広げて選び直す。

秘密を、どこに保管し、どう配るか。取り出せない形で預ける鍵（GitHub App の private key）は範囲外で、[ADR-0005](0005-choose-how-workflows-authenticate-to-github-beyond-github-token.md) が決める。

## Decision Drivers

* 秘密の置き場と、誰が読めるかを、コード（宣言）で管理できること
* 1つの秘密を、CI・クラスタ・手元が同じ実体から読めること
* 実体が1箇所で、入れ替えが1回で済み、何がどこにあるかを一覧できること

## Considered Options

* AWS Parameter Store に置き、CI は IAM OIDC、クラスタは External Secrets Operator、手元は AWS のログインで読む
* GitHub の repository secret に置く（クラスタと手元は、それぞれの仕組みに別に置く）

## Decision Outcome

Chosen option: "AWS Parameter Store に置き、CI は IAM OIDC、クラスタは External Secrets Operator、手元は AWS のログインで読む", because 誰が読めるかを IAM の trust policy と policy として、置き場を parameter の名前として、どの秘密も同じ形のコードで宣言できる。値は Terraform を通さずに書けるので、state に平文が載らない。実体は Parameter Store の1つに集まり、CI・クラスタ・手元のどれもが同じ実体を読むので、入れ替えはその1つを書き換えるだけで済み、何がどこにあるかも1箇所で一覧できる。AWS は鍵を預けるために既に使っている（ADR-0005）ので、置き場のために新しいアカウントを足さない。

GitHub の repository secret は、どの repo のどの workflow が読めるかを宣言する場所が無く、読めるのはその repo の workflow 全体になる。値を Terraform（`github_actions_secret`）で書けば管理はコードに寄るが、値が state に平文で載る。書き込み専用で手元から読めないので、CI と手元の両方が使う秘密は別に持つしかなく、クラスタの秘密も別の仕組みになる。置き場が repo と使う場所の数だけ分かれ、入れ替えもその数だけ要り、一覧もできない。

### Consequences

* Good, because 秘密を足すときも入れ替えるときも、使う場所によらず同じ手順になる
* Good, because 読まれた事実が CloudTrail に残る
* Bad, because ジョブの実行中は runner の上に秘密が載る。workflow の中の悪意あるコードからは、repository secret と同じく守れない
* Bad, because AWS のアカウントが、すべての秘密を握る一点になる
* Bad, because 読み出しの経路が AWS に寄るので、それが壊れるとすべての消費側が同時に止まる
* Bad, because 手元で秘密を使う作業の前に、AWS へのログインが要る

## More Information

* この決定は [ADR-0004](0004-choose-the-home-for-secrets-shared-across-repositories.md) を置き換える
