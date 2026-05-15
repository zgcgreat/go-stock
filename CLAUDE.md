# Project Workflow

These workflow instructions are explicit opt-in. Do not apply them by default. Only use them when the user explicitly asks for the workflow or invokes its command.

Prefer the `openspec-superpowers-workflow` command when a feature should start with OpenSpec artifacts, then use Superpowers for implementation planning, TDD, and verification.

OpenSpec is only the artifact-locking stage in this combined workflow. After `tasks.md` is complete, do not continue with OpenSpec apply-style implementation. Hand off to Superpowers execution for planning, TDD, and fresh verification.

Do not stop with a readiness prompt such as "run /opsx:apply" or "let me start implementation". Unless the user explicitly asked to pause after OpenSpec artifacts, continue directly into Superpowers execution by writing the implementation plan.

If `.superpowers-memory/` exists in the repository, treat it as shared project memory and keep it up to date during the workflow.


<!-- superpowers-memory:start -->
## Superpowers Memory

If `.superpowers-memory/` exists in this repository, read these files at the start of each session before asking for project background:

1. `.superpowers-memory/PROJECT_CONTEXT.md`
2. `.superpowers-memory/CURRENT_STATE.md`
3. `.superpowers-memory/DECISIONS.md` when it exists
4. `.superpowers-memory/KNOWN_FAILURES.md` when it exists
5. `.superpowers-memory/VERIFICATION_BASELINE.md` when it exists
6. `.superpowers-memory/TEAM_PREFERENCES.md` when it exists
7. `.superpowers-memory/USER_PROFILE.md` when it exists
8. `.superpowers-memory/AGENT_NOTES.md` when it exists
9. The newest files under `.superpowers-memory/session-journal/`

Use them to recover project context, recent decisions, active work, verification expectations, team preferences, durable user preferences, agent-side execution reminders, and likely next steps.

Before ending a meaningful Superpowers-related session, update:

- `.superpowers-memory/CURRENT_STATE.md`
- any durable files that changed during the work, especially `DECISIONS.md`, `KNOWN_FAILURES.md`, `VERIFICATION_BASELINE.md`, `TEAM_PREFERENCES.md`, `USER_PROFILE.md`, and `AGENT_NOTES.md`
- one short markdown note under `.superpowers-memory/session-journal/`

When memory updates are part of the workflow, run `scripts/validate-superpowers-memory.ps1` before claiming completion.

Do not treat memory files as permission to auto-enable Superpowers workflows. Workflow activation remains explicit opt-in.
<!-- superpowers-memory:end -->

