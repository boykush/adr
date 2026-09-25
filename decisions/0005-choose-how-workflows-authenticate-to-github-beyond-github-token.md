---
status: accepted
date: 2026-09-24
---

# Choose how workflows authenticate to GitHub beyond GITHUB_TOKEN

## Context and Problem Statement

GITHUB_TOKEN は実行中のリポジトリに閉じ、それで起こした操作は次の workflow を起動しない。これを超える操作——他のリポジトリへの書き込みや、required check を走らせる PR の作成——をする workflow が、複数のリポジトリにある。

資格は PAT か GitHub App になる。App のインストールトークンは、App の private key で署名した JWT と引き換えに発行される。App の標準の手段は公式の `actions/create-github-app-token` で、private key を repository secret から値として受け取る。

**App の private key には期限が無い**。repository secret に置いた鍵は、一度読まれれば、App から revoke するまで誰でもトークンを発行できる。トークンは1時間で切れるが、元になる鍵は切れない。GitHub も、鍵は失効せず手で revoke するものだとしたうえで、key vault に入れて署名専用にすることを勧めている。同じ対策は Flatt Security の連載でも挙がっている。

<https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps>

<https://docs.github.com/en/apps/creating-github-apps/about-creating-github-apps/best-practices-for-creating-a-github-app>

<https://blog.flatt.tech/entry/2026-github-actions-security-part2>

GITHUB_TOKEN で足りない操作を、どの資格で行い、その資格をどう持つか。

## Decision Drivers

* 漏れたものが、期限か権限の取り消しで無効になること
* 操作の主体が owner 本人と分かれ、権限を用途ごとに絞れること
* どのリポジトリが何として振る舞えるかを列挙でき、剥がせること
* どの run がトークンを取ったかを追えること
* 消費側の変更が、トークンを取る step の差し替えで済むこと
* 守るべきアカウントと信頼の経路を増やさないこと

## Considered Options

* GitHub App の鍵を AWS KMS に入れ、署名だけを任せる
* GitHub App の鍵を Azure Key Vault に入れ、署名だけを任せる
* GitHub App の鍵を repository secret に置き、公式の `actions/create-github-app-token` に渡す
* PAT を repository secret に置く

## Decision Outcome

Chosen option: "GitHub App の鍵を AWS KMS に入れ、署名だけを任せる", because 鍵が KMS から出ず、run が手にするのは App の JWT への署名だけになる。漏れうるのは1時間で切れるトークンと、IAM で剥がせる「署名を頼める」ことに限られる。トークンを取る既製の Action があり、公式の action と入出力が揃っているので、消費側はトークンを取る step を差し替えるだけで移れる。PAT はそれ自体が資格なので署名だけを任せられず、owner 本人として振る舞う。公式の action は、置き場がどこでも鍵を値で受け取る。Azure Key Vault も同じ形を作れるが、守るべきアカウントと OIDC の信頼が増え、トークンを取る既製の Action も無い。

### Consequences

* Good, because 操作の主体が App ごとに owner 本人と分かれ、権限を用途ごとに絞れる
* Good, because 署名できるリポジトリが IAM に列挙され、どの run が署名したかが CloudTrail に残る
* Bad, because 全リポジトリを管理する App の経路に、出て間もない第三者の Action が入る
* Bad, because `id-token: write` は job 単位なので、job の間はどの step もトークンを取り直せる
* Bad, because AWS のアカウントが、すべての App の署名を握る一点になる
* Bad, because 自分で JWT に署名する consumer は KMS を使えず、その鍵は取り出せる形で残る

## More Information

* KMS で署名してトークンを取る既製の Action: <https://github.com/suzuki-shunsuke/create-github-app-token-aws-kms>
