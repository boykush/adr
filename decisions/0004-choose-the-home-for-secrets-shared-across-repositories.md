---
status: accepted
date: 2026-09-23
---

# Choose the home for secrets shared across repositories

## Context and Problem Statement

複数のリポジトリで同じ秘密が要る。Claude Code Actions のトークンがその最初の例で、`@claude` を動かすリポジトリの数だけ同じ値を配ることになる。

owner は User アカウントなので **organization secret が無い**。GitHub 側に共有する手段が存在せず、リポジトリごとに secret を置くしかない。実際 wiki は `CLAUDE_CODE_OAUTH_TOKEN`、scraps は `ANTHROPIC_API_KEY` と、同じ用途の秘密が別の名前と別の課金経路に割れていた。

この形は2つ壊れている。**入れ替えがリポジトリの数だけ要る**ので現実には回らない。そして**長命な秘密が GitHub 側に N 個残り**、どこに何が置かれているかを一覧する手段も無い。

跨いで使う秘密を、どこに置くか。

## Decision Drivers

* 秘密の実体が1箇所で、入れ替えが1回で済むこと
* リポジトリ側に長命な秘密を残さないこと
* 誰が読めるかを宣言として列挙でき、読まれた事実を追えること
* 消費側に足す物が最小で、リポジトリごとの設定を増やさないこと
* 追加費用がほぼ無いこと

## Considered Options

* AWS（Parameter Store + IAM OIDC）
* Google Cloud（Secret Manager + Workload Identity 連携）
* リポジトリごとの GitHub secret（現状維持）

## Decision Outcome

Chosen option: "AWS（Parameter Store + IAM OIDC）", because 秘密は Parameter Store の1本に集まり、各リポジトリは**その run の OIDC で読む**。GitHub 側に残る秘密が無くなり、入れ替えは1コマンドになる。誰が読めるかは IAM role の trust policy に列挙され、読まれた事実は CloudTrail に残る。standard tier は保管も API 呼び出しも無料で、費用は実質増えない。

Google Cloud も同じ構図を作れるが、`google-github-actions/auth` が認証情報ファイルを `$GITHUB_WORKSPACE` に書く。Claude が commit するジョブでは `.gitignore` の1行を消費側すべてに足すことになり、**リポジトリごとの設定を増やさない**という動機と逆を向く。避けるには STS の交換を自分で書くしかない。AWS 側は `configure-aws-credentials` が既定で環境変数だけを使い、`output-env-credentials: false` を付ければ**認証情報をジョブの環境にすら置かず**読み取りステップにだけ渡せる。値の読み出しに要る追加のアクションも無い——AWS CLI は runner に同梱されている。

消費側は composite action を呼ぶ。秘密ごとに読み出しの手順を書き散らさず、role・パラメータ名・マスクの扱いを1箇所に閉じるため。リポジトリを増やす操作は、trust policy に1行足すことと、ワークフローに2ステップ足すことだけにする。

鍵そのものを取り出せない形で預ける話（GitHub App の private key と KMS）は、この決定の範囲外。

### Consequences

* Good, because 入れ替えが1回で済むので、現実に回せる頻度になる
* Good, because GitHub 側に長命な秘密が残らず、読める主体が IAM に列挙され、読まれた事実が CloudTrail に残る
* Bad, because ジョブの実行中は runner の上に秘密が載る。Claude のステップが任意コードを実行する以上、認証ステップと実行ステップを別ジョブに分ける定石が使えない
* Bad, because クラウドが1つ増える。請求とアカウントの管理先が DigitalOcean・Cloudflare・HCP に続いて4つ目になる
* Bad, because composite action が壊れるとすべての消費側が同時に止まる
* Bad, because 消費側は composite action を SHA で固定するので、変更のたびに各リポジトリの pin を上げることになる
