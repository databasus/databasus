# Verification agent

The verification agent is part of the backend. Its code rules are in the "Verification agent" section of [`backend/AGENTS.md`](../../backend/AGENTS.md). Its place in the system is described in [`docs/architecture.md`](../../docs/architecture.md#verification-agent). This module is moving into `backend/cmd/verification-agent` and `backend/internal/agent`. Until it moves, `mise run agent:lint`, `agent:format`, `agent:test` and `agent:e2e` check it.
