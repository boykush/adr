---
status: accepted
date: 2026-09-20
---

# Choose the project instruction file for coding agents

## Context and Problem Statement

Claude Code と Codex を併用している。Codex は AGENTS.md を読む。Claude Code はこれまで CLAUDE.md しか読まなかったが、v2.1.277 から CLAUDE.md が無ければ AGENTS.md を読む。

<https://code.claude.com/docs/en/memory#agents-md>

エージェント向けのプロジェクト指示を、どのファイルに書くか。

## Decision Drivers

* 特定のエージェントに縛られないこと
* 置くファイルが1つで済むこと

## Considered Options

* AGENTS.md だけを置き、CLAUDE.md は置かない
* AGENTS.md を実体にし、CLAUDE.md から import かシンボリックリンクで繋ぐ
* CLAUDE.md だけを置く

## Decision Outcome

Chosen option: "AGENTS.md だけを置き、CLAUDE.md は置かない", because CLAUDE.md が無ければ Claude Code も AGENTS.md を読むので、CLAUDE.md を置く理由がなくなった。`.claude/CLAUDE.md` と `CLAUDE.local.md` も、あると AGENTS.md が読まれなくなるので置かない。

### Consequences

* Good, because AGENTS.md 1つで Claude Code と Codex の両方に指示が届く
* Bad, because v2.1.277 未満の Claude Code や、Bedrock 経由・テレメトリ無効のセッションでは指示が読まれない
