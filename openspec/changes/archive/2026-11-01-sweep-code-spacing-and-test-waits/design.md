## Context

See `proposal.md` for the motivation. `AGENTS.md` ("Blank lines separate logical steps") defines the spacing rule, and `.agent-hooks/code-shape-prompt.md:18` reviews it only in code a diff added or changed. An example of the problem is `backend/internal/app/apptest/physicaltest/promotion.go:64`, where a whole scenario runs as one block.

A parser-based count, run on 2026-11-01, found 942 of the 2335 files holding a block that could be a wall, as the decisions below define it. These are candidates only: six assertions in a row can be one step.

The time limits to change are all in `frontend/tooling/runBrowserE2E.test.ts`: the per-test limits at `:91`, `:110` and `:262`, the per-wait limit at `:225`, and the request limit at `:51`. `frontend/vitest.browser-e2e.config.ts:7` inherits the shared 60-second test limit from `frontend/vitest.config.ts`.

Another agent is working in the main checkout. It has 24 uncommitted paths. Of these, 21 are on the sweep's list, including `frontend/vitest.config.ts` with its setup-file entry. The other three are the new `frontend/src/shared/lib/test/conditionWait.ts` and the uncommitted wording of the rule against per-wait limits in `.agents/skills/testing/SKILL.md` and `docs/testing.md`.

The approach is constrained by the root `AGENTS.md` sections on blank lines, tests and other agents' changes, by the blank-line rule links in `backend/AGENTS.md:96` and `frontend/AGENTS.md:79`, and by `website/AGENTS.md` for the website files. It changes no architecture, so `docs/architecture.md` stays as it is.

## Goals / Non-Goals

**Goals:**

- A reader can see the steps in code that was a wall of statements.
- Proof for every swept file that only empty lines were added.
- The browser lifecycle suite follows the rule against limits on one wait or one test.

**Non-Goals:**

- Full compliance with the blank-line rule. Short mixed blocks stay until a later change touches them.
- Consistent spacing across the code base.

## Decisions

### What counts as a wall

A block gets blank lines when it has at least six statements or about twelve solid lines, and those statements mix at least two distinct steps, such as setup, main work and assertions. A shorter block stays, even with mixed steps. So does a long block of one kind, such as a list of assertions, field assignments or registrations.

Rejected: applying the full `AGENTS.md` rule, which would turn the sweep into micro-fixes the user explicitly ruled out. Rejected: a pure statement-count threshold, which misses a few long multi-line statements and splits long uniform lists.

### Every file is covered, candidates are read

The sweep covers the whole list, as the user chose. A file reaches an agent when a parser finds a block in it with at least six statements, or at least two statements spanning ten or more lines with no blank line between them. Ten leaves a margin under the criterion's "about twelve". A file with no such block cannot receive an edit, so reading it changes nothing. That gives 942 files, with the same edits at about 40% of the reads.

Rejected: reading all 2335 files, which costs more and cannot change the result under this criterion. Rejected: counting only blocks of six statements, which misses a wall of a few long statements.

### Only additions, proved in two ways

Sweep agents add empty lines and change nothing else. A throwaway checker outside the repository accepts a file only when both checks hold:

- every changed line in `git diff` of the file is an added empty line;
- the file's tokens equal those of its `HEAD` version, using `go/scanner` for Go, `tokenize` for Python, and for TypeScript the tokens of a `ts.createSourceFile` parse, including template and JSX text. A bare TypeScript scanner is not enough: it lexes template text after a `${...}` substitution as ordinary code, so it would miss a blank line there.

The token check is needed because a blank line inside a raw or template string looks like an allowed addition but changes the program, as in the script embedded in `frontend/tooling/runChild.test.ts`.

Rejected: `git diff --ignore-blank-lines` alone, which cannot see a blank line inside a string. Rejected: a committed checker, because no later work needs it. Rejected: an automatic formatter or linter fix such as `wsl` or `padding-line-between-statements`, because where one step ends depends on meaning and these tools add a blank line by syntax, against the rule's "do not put a blank line after every statement".

### Verification without the backend test suite

For the sweep, the checker's proof replaces package tests: identical tokens compile to the same program, apart from line numbers. `mise run backend:lint` type-checks the Go code, including tests. The frontend suite is fast enough to run in full.

Rejected: running the touched backend packages, because the list holds all 1594 backend source files, so that run equals the full suite, which runs only when the user asks. A test that depends on line numbers is the one gap, covered under Risks.

### Sonnet agents on disjoint lists in the main checkout

Background agents on Sonnet at medium effort each get a batch of up to about 10,000 lines, grouped by module, with at most six running at once. They edit only their own listed files and run no git-changing command and no formatter. The checker runs on each returned batch, and a batch that fails goes back to an agent. Before the checker, a throwaway filter rebuilds each file from `HEAD` and keeps only insertions inside a block that meets the size limit in the wall criterion, read in the filter as six statements, or three statements spanning twelve lines. Agents still judge whether the steps are mixed, but they cannot drift into micro-fixes on small blocks.

Rejected: one worktree per agent. Each worktree would need a commit to bring its work back, and commits wait for the user's pack request. Opus at low effort also fits but costs more per file, and the checker already rules out logic errors.

### Other agents' files wait

Before each batch and again at the end, the sweep removes any listed file with uncommitted changes it did not make. These files return to the list when clean. Any file still dirty at the end is left alone and named in the report.

Rejected: editing them now, which `AGENTS.md` forbids and which would mix two agents' work in one diff.

### The two waits files stay out of the sweep

`frontend/tooling/runBrowserE2E.test.ts` and `frontend/vitest.browser-e2e.config.ts` change real code in the waits work, so the checker would always reject them. They leave the sweep's list, and the waits work applies the wall criterion to them directly.

Rejected: comparing them against a snapshot taken after the waits work, which adds a second baseline to the checker for two files.

### One limit for the browser lifecycle suite

`frontend/vitest.browser-e2e.config.ts` sets `testTimeout: 180_000` for its suite, which holds real browser and container runs, and the three per-test limits go away. The wait at `:225` moves to `waitForCondition`, which the test imports from `../src/shared/lib/test/conditionWait`. The check at `:51` drops its abort signal and asserts `ECONNREFUSED`, so a hung gateway fails the test instead of passing it.

The kill deadline at `:29` and the command limits at `:143` and `:217` stay, because they stop a child process rather than pace the test.

Rejected: keeping per-test limits with a comment, because the rule forbids them. Rejected: raising the shared 60-second limit in `frontend/vitest.config.ts`, which would also loosen the bound on every screen test. `conditionWait.ts` is the other agent's uncommitted file. The working tree can run the import today, but the commit of this change has to land after that agent's commit. `docs/testing.md` gets one sentence naming the suite's limit once that agent has committed the file. If it is still uncommitted at the end, the sentence is left out and named in the report.

## Risks / Trade-offs

- [The diff touches hundreds of files, and open branches may conflict around added lines] → The two branches in `.claude/worktrees` have no commits ahead of `develop`, and other open branches were not checked. Pack the change soon after it is accepted.
- [Sonnet draws a step boundary badly] → The result is cosmetic. Implementation review judges it on changed lines, and the checker rules out behavior changes.
- [Reviewers report short blocks next to the edits] → The wall criterion above decides. Such a finding is rejected with that reason, or treated as `minor`.
- [A test depends on source line numbers] → Search found three `runtime.Caller(0)` calls in backend test code, in the mTLS fixtures and `backend/internal/lib/testcontainers/ssh_bastion.go:54`, and all use the file path, never the line. `pnpm --dir frontend test` covers the frontend.
- [Another agent commits or edits a listed file during the sweep] → The checker reports a non-blank diff for that file. The file is then set aside as above.
- [The browser lifecycle suite needs Docker and is slow] → It runs once for the waits task, not for the sweep.
