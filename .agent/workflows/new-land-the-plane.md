---
description: Automates the session wrap-up protocol, including testing, git sync, and logging to Obsidian.
---

## 6. Session Completion Protocol (Landing the Plane)

**MANDATORY:** You cannot sign off until work is **Committed** and **Tracked**. We are operating in **Local Mode**, so there is no remote push.

**CRITICAL:** Complete ALL subsections (6.1-6.8) before ending your turn. Do NOT provide a Session Summary unless you have executed steps 6.1-6.7. A Session Summary without completing the full protocol is a failed handoff.

### 6.1 Capture Orphans

Create `bd` issues for any remaining work, bugs, or refactors discovered during the session.

### 6.2 Update Knowledge

If you solved a tricky error or established a new pattern, append a brief note to `LESSONS.md`.

If model performance was notably good or bad (unexpected for its tier), add an observation to `docs/LLM_CAPABILITIES.md`.

### 6.3 Run Quality Gates

```bash
npm run lint
```

```bash
npm run test # if tests exist for changed code
```

Fix failures immediately. Do not "disavow" them or leave them for the next agent.

### 6.4 Validate at Runtime

**CRITICAL:** This step is NOT optional. See Section 5.1 for the full verification protocol.

For CLI commands: run `--help`, `--dry-run`, and real execution with data verification.

### 6.5 Update Issue Status

- **Finished:** `bd close <id>` (with closing comment)
- **Incomplete handoff:** `bd update <id> --status todo` with explanatory comment

### 6.6 Commit Sequence (Local Mode)

```bash
bd sync                                    # Synchronize local database
git add -A                                 # Stage code, docs, AND .beads/
git commit -m "feat(bd-<id>): <description>"  # Reference task ID in commit
# DO NOT PUSH. We are operating in Local Mode.
```

*Rationale:* Including `.beads/issues.jsonl` in the feature commit keeps task state and code changes atomic.

### 6.7 Merge & Cleanup

```bash
git checkout main
git merge --no-ff issue/bd-<id>-<short-desc> -m "Merge branch 'issue/bd-<id>-<short-desc>'"  # ALWAYS use -m to avoid editor
git branch -d issue/bd-<id>-<short-desc>        # Delete feature branch
```

**CRITICAL:** Always use `-m "message"` with `git merge` to avoid opening an editor. Agents cannot interact with vim/vi.

### 6.8 Final Handoff — Session Summary (REQUIRED)

**DO NOT write this summary until you have completed steps 6.1-6.7.**

Provide this structured block at the end of your final message. The next agent will rely on it.

```markdown
## Session Summary

### Completed This Session
- [bullet list of what was done, with file paths and task IDs]

### Current State
- Branch: `<branch-name>` (or `main` if merged)
- Git status: clean / dirty (list uncommitted files if dirty)
- In-progress tasks: none / <task-id> (should be none at session end)

### Next Steps
- [what the next agent should do, or "none—all tasks closed"]

### Blockers / Warnings
- [any issues, gotchas, or context the next agent needs]
```

**Log Session to Obsidian**
   - **Agent Action**: Identify the current repository/workspace name (e.g., `obsidian-agent`).
   - **Agent Action**: Format the log entry as: `[{WorkspaceName}] {Summary}`.
   - **Agent Action**: Determine today's date (YYYY-MM-DD).
   - **Agent Action**: Use the `obsidian-cli` CLI tools to log this summary.
     - Call `read_note` for `devlog/{YYYY-MM-DD}.md`.
       - If it fails (note doesn't exist), start with the summary as content.
       - If it succeeds, append the summary to the existing content.