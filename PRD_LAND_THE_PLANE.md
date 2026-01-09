# Product Requirements Document: "Land the Plane" Automated Workflow

## 1. Executive Summary
This document outlines the requirements for an automated AI agent workflow triggered by the command "land the plane". The workflow's primary function is to synthesize the work performed during a coding session and append a concise one-line summary to a daily development log file in the user's Obsidian vault.

## 2. User Story
**As a** developer using the Antigravity IDE,
**I want** to issue a single command ("land the plane") when I finish a task,
**So that** a concise summary of my work is automatically logged in my daily devlog without manual context switching.

## 3. Functional Requirements

### 3.1 Trigger
- The workflow is initiated by the user explicitly invoking the phrase **"land the plane"** (or a corresponding slash command, e.g., `/land-the-plane`).

### 3.2 Inputs
- **Session Context**: The agent must have access to the recent conversation history, tool outputs, and modified files to generate the summary.
- **Tools**: Access to `obsidian-mcp` tools.

### 3.3 Process Steps
1.  **Summarization**: 
    - Identify the current workspace/repository name.
    - Analyze the session to generate a concise summary.
    - Format: `[{WorkspaceName}] {Summary}` (e.g., `[obsidian-agent] Implemented GoogleAuth provider`).
2.  **Date Resolution**: Determine the current date in `YYYY-MM-DD` format.
3.  **Target Resolution**: Construct the target file path: `devlog/{YYYY-MM-DD}.md`.
4.  **Content Appending**:
    - Check if the file exists using `read_note`.
    - If it exists, append the new summary line to the existing content.
    - If it does not exist, create the file with the summary line as the initial content.
    - Write the updated content back to the vault using `create_note` (acting as an upsert/overwrite with new content).

### 3.4 Outputs
- A modified markdown file in the `devlog/` folder of the user's vault containing the new summary entry.
- A confirmation message to the user: "Landed the plane: [Summary]".

## 4. Technical Specifications

### 4.1 Tools Required
The following Obsidian MCP tools will be utilized:
- **`read_note(path)`**: To retrieve the current content of the devlog (for appending).
- **`create_note(path, content)`**: To write the updated content (original + new summary) back to the file.
- **`search_notes(query)`** (Optional): To verify folder structure if needed.

### 4.2 Handling "Append" Logic
Since the generic `append_to_file` tool might not exist or `append_to_daily_note` might target a different default folder:
- The agent will implement a **Read-Modify-Write** pattern:
    1.  `current_content = read_note("devlog/2024-01-01.md")`
    2.  `new_content = current_content + "\n- " + summary`
    3.  `create_note("devlog/2024-01-01.md", new_content)`

### 5. Future Enhancements
- Integration with `append_to_daily_note` if the `devlog` folder becomes the default daily note location.
- Automatic git commit message generation based on the same summary.
- Posting the summary to a team Slack/Discord via another MCP tool.
