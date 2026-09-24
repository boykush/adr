---
status: accepted
date: 2026-09-24
---

# Choose how workflows authenticate to GitHub beyond GITHUB_TOKEN

## Context and Problem Statement

GITHUB_TOKEN は実行中のリポジトリに閉じ、それで起こした操作は次の workflow を起動しない。これを超える操作——他のリポジトリへの書き込みや、required check を走らせる PR の作成——をする workflow が、複数のリポジトリにある。

資格は PAT か GitHub App になる。PAT はそれ自体がトークンで、owner 本人として振る舞う。App のインストールトークンは、App の private key で署名した JWT と引き換えに発行される。App の標準の手段は公式の `actions/create-github-app-token` で、private key を repository secret から値として受け取る。

**App の private key には期限が無い**。repository secret に置いた鍵は、一度読まれれば、App から revoke するまで誰でもトークンを発行できる。トークンは1時間で切れるが、元になる鍵は切れない。GitHub も、鍵は失効せず手で revoke するものだとしたうえで、key vault に入れて署名専用にすることを勧めている。

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

Chosen option: "GitHub App の鍵を AWS KMS に入れ、署名だけを任せる", because 鍵が KMS から出ず、run が手にするのは App の JWT への署名だけになる。漏れうるのは1時間で切れるトークンと、IAM で剥がせる「署名を頼める」ことに限られる。PAT はそれ自体が資格なので署名だけを任せられず、owner 本人として振る舞う。公式の action は、置き場がどこでも鍵を値で受け取る。Azure Key Vault も同じ形を作れるが、守るべきアカウントと OIDC の信頼が増え、トークンを取る既製の Action も無い。

* GITHUB_TOKEN で足りない操作は GitHub App として行い、PAT は使わない
* トークンは `suzuki-shunsuke/create-github-app-token-aws-kms` で取る。公式の action と入出力が揃っていて、`private-key` を `kms-key-id` と `role-to-assume` に替えれば移れる
* KMS の key と role は App ごとに分け、role に許すのは RS256 の `kms:Sign` だけにする。束ねると、弱い App を使うリポジトリの run が、全リポジトリを管理する App としても署名できる
* trust policy にはその App を使うリポジトリを列挙し、owner 全体のワイルドカードにはしない
* 鍵の material は state に載るので Terraform を通さず、手元から CLI で入れて、手元の PEM は消す。apply する CI の role には、署名も material の投入も key policy の変更も許さない
* workflow の外で自分で JWT に署名する consumer（Argo CD Image Updater など）は KMS を使えない。その鍵は Parameter Store に置き、consumer へ渡す workflow が OIDC で読む

<https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps>

<https://github.com/suzuki-shunsuke/create-github-app-token-aws-kms>

### Consequences

* Good, because 鍵が GitHub からも runner からも消え、漏れうるのは1時間で切れるトークンになる
* Good, because 操作の主体が App ごとに owner 本人と分かれ、権限を用途ごとに絞れる
* Good, because 署名できるリポジトリが IAM に列挙され、どの run が署名したかが CloudTrail に残る
* Bad, because 全リポジトリを管理する App の経路に、出て間もない第三者の Action が入る
* Bad, because `id-token: write` は job 単位なので、job の間はどの step もトークンを取り直せる
* Bad, because AWS のアカウントが、すべての App の署名を握る一点になる
* Bad, because 自分で JWT に署名する consumer の鍵は、取り出せる形で残る
