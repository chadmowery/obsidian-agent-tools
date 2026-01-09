---
description: Automates the session wrap-up protocol, including testing, git sync, and logging to Obsidian.
---

# Land the Plane Protocol

1. **Run Quality Gates**
   - Run the full test suite to ensure stability.
   ```bash
   go test ./...
   ```
   - Verify the main binaries build.
   ```bash
   go build -o /dev/null ./cmd/obsidian-mcp
   go build -o /dev/null ./cmd/obsidian-mcp-http
   ```

2. **Check Issue Status**
   - List the user's current issues.
   ```bash
   bd show
   ```
   - **User Action**: Please review the above issues. Use `bd close <id>` to close completed work or `bd create` to file new issues for outstanding items.

3. **Push to Remote**
   - Execute the mandatory git sync sequence.
   ```bash
   git pull --rebase
   bd sync
   git push
   ```

4. **Log Session to Obsidian**
   - **Agent Action**: Identify the current repository/workspace name (e.g., `obsidian-agent`).
   - **Agent Action**: Generate a concise one-line summary of the work completed in this session.
   - **Agent Action**: Format the log entry as: `[{WorkspaceName}] {Summary}`.
   - **Agent Action**: Determine today's date (YYYY-MM-DD).
   - **Agent Action**: Use the `obsidian-remote` MCP server tools to log this summary.
     - Call `read_note` for `devlog/{YYYY-MM-DD}.md`.
       - If it fails (note doesn't exist), start with the summary as content.
       - If it succeeds, append the summary to the existing content.
     - Call `create_note` (or `append_to_daily_note` if appropriate) with the updated content.
   - **User Notification**: "Plane landed. Summary logged: [Summary]"
