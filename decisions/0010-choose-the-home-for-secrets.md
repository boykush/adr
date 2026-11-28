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

* 使う場所や共有かどうかに関わらず、保管と配り方が1つの方式で、コード（宣言）で管理できること
* 1つの秘密を、CI・クラスタ・手元が同じ実体から読めること
* 実体が1箇所で、入れ替えが1回で済み、何がどこにあるかを一覧できること
* 誰が読めるかを宣言として列挙でき、読まれた事実を追えること
* 追加費用がほぼ無いこと

## Considered Options

* AWS Parameter Store に置き、CI は IAM OIDC、クラスタは External Secrets Operator、手元は AWS のログインで読む
* Google Cloud Secret Manager に置き、同じ構図で読む
* 使う場所ごとの仕組みに置く（CI は repository secret、クラスタは Kubernetes の Secret、手元は各ツールの設定）

## Decision Outcome

Chosen option: "AWS Parameter Store に置き、CI は IAM OIDC、クラスタは External Secrets Operator、手元は AWS のログインで読む", because 秘密の実体は Parameter Store の1つに集まり、CI・クラスタ・手元のどれもが同じ実体を読む。置き場と、誰が読めるか（IAM の trust policy と policy）は、同じ形のコードで宣言でき、入れ替えはその1つを書き換えるだけで済む。読まれた事実は CloudTrail に残る。standard tier は保管も API 呼び出しも無料で、鍵を預ける AWS（ADR-0005）と同じアカウントに収まるので、守るアカウントも増えない。

Google Cloud も同じ構図を作れるが、`google-github-actions/auth` が認証情報ファイルを `$GITHUB_WORKSPACE` に書くので、消費側すべての `.gitignore` に1行を足すことになる。鍵を預ける AWS とは別に、守るアカウントと OIDC の信頼も1つ増える。

使う場所ごとの仕組みに置くと、複数の場所が読む秘密は写しになり、入れ替えが写しの数だけ要る。repository secret は手元から読めないので、手元で使う分は別に持つしかない。置き場が場所ごとに違うので、一覧も、読めるものの宣言も1つの形にならない。

### Consequences

* Good, because 秘密を足すときも入れ替えるときも、使う場所によらず同じ手順になる
* Bad, because ジョブの実行中は runner の上に秘密が載る。workflow の中の悪意あるコードからは、repository secret と同じく守れない
* Bad, because AWS のアカウントが、すべての秘密を握る一点になる
* Bad, because 読み出しの経路が AWS に寄るので、それが壊れるとすべての消費側が同時に止まる
* Bad, because 手元で秘密を使う作業の前に、AWS へのログインが要る

## More Information

* この決定は [ADR-0004](0004-choose-the-home-for-secrets-shared-across-repositories.md) を置き換える
