---
adr_id: "0001"
comments:
    - author: boykush
      comment: "1"
      date: "2026-09-20 11:39:30"
status: decided
title: Choose the project instruction file for coding agents
---

## <a name="question"></a> Question

Claude Code と Codex を併用している。Codex は AGENTS.md を読む。Claude Code はこれまで CLAUDE.md しか読まなかったが、v2.1.277 から CLAUDE.md が無ければ AGENTS.md を読む。

<https://code.claude.com/docs/en/memory#agents-md>

エージェント向けのプロジェクト指示を、どのファイルに書くか。

## <a name="options"></a> Options

1. <a name="option-1"></a> AGENTS.md だけを置き、CLAUDE.md は置かない
2. <a name="option-2"></a> AGENTS.md を実体にし、CLAUDE.md から import かシンボリックリンクで繋ぐ
3. <a name="option-3"></a> CLAUDE.md だけを置く

## <a name="criteria"></a> Criteria

- 特定のエージェントに縛られないこと
- 置くファイルが1つで済むこと

## <a name="outcome"></a> Outcome
We decided for [Option 1](#option-1) because: CLAUDE.md が無ければ Claude Code も AGENTS.md を読むので、CLAUDE.md を置く理由がなくなった。`.claude/CLAUDE.md` と `CLAUDE.local.md` も、あると AGENTS.md が読まれなくなるので置かない。

### Consequences

- AGENTS.md 1つで Claude Code と Codex の両方に指示が届く
- v2.1.277 未満の Claude Code や、Bedrock 経由・テレメトリ無効のセッションでは指示が読まれない

## <a name="comments"></a> Comments
<a name="comment-1"></a>1. (2026-09-20 11:39:30) boykush: marked decision as decided
