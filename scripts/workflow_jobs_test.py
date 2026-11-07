import textwrap
import unittest

from workflow_jobs import get_jobs


class WorkflowJobsTest(unittest.TestCase):
    def test_preserves_inline_steps_and_job_configuration_without_aliases(self) -> None:
        workflow = textwrap.dedent("""\
            name: Example

            jobs:
              check:
                env:
                  ARTIFACT_NAME: candidate-${{ matrix.arch }}
                strategy:
                  matrix:
                    arch: [amd64, arm64]
                steps:
                  - name: Check
                    run: echo checked
            """)

        jobs = get_jobs(workflow)

        self.assertEqual(jobs["check"], workflow.split("  check:\n")[1])

    def test_expands_complete_steps_at_each_call_site_with_caller_configuration(self) -> None:
        workflow = textwrap.dedent("""\
            name: Example

            jobs:
              first:
                env:
                  ARTIFACT_NAME: first
                steps:
                  - &download
                    name: Download
                    id: download
                    continue-on-error: true
                    with:
                      name: ${{ env.ARTIFACT_NAME }}

                  - name: Check
                    run: echo checked

              second:
                env:
                  ARTIFACT_NAME: second
                steps:
                  - *download

              third:
                steps:
                  - *download
            """)

        jobs = get_jobs(workflow)

        for job_name in ("first", "second", "third"):
            with self.subTest(job=job_name):
                self.assertIn("      - name: Download\n        id: download\n", jobs[job_name])
                self.assertIn("        continue-on-error: true\n", jobs[job_name])
                self.assertIn("          name: ${{ env.ARTIFACT_NAME }}\n", jobs[job_name])
                self.assertNotIn("&download", jobs[job_name])
                self.assertNotIn("*download", jobs[job_name])
        self.assertIn("ARTIFACT_NAME: first", jobs["first"])
        self.assertIn("ARTIFACT_NAME: second", jobs["second"])
        self.assertIn("- name: Check", jobs["first"])
        self.assertNotIn("- name: Check", jobs["second"])

    def test_preserves_multiline_shell_and_expands_an_alias_at_end_of_file(self) -> None:
        workflow = textwrap.dedent("""\
            name: Example

            jobs:
              first:
                steps:
                  - &prepare
                    name: Prepare
                    run: |
                      cd backend
                      go mod download
              second:
                steps:
                  - *prepare""")

        jobs = get_jobs(workflow)

        self.assertEqual(jobs["first"], jobs["second"])
        self.assertIn(
            "        run: |\n          cd backend\n          go mod download\n", jobs["second"]
        )

    def test_rejects_missing_duplicate_forward_and_unsupported_anchors(self) -> None:
        for steps, diagnostic in (
            ("      - *missing\n", "Unknown step anchor: missing"),
            (
                "      - *later\n      - &later\n        name: Later\n        run: echo later\n",
                "Unknown step anchor: later",
            ),
            (
                "      - &duplicate\n        name: First\n"
                "      - &duplicate\n        name: Second\n",
                "Duplicate step anchor: duplicate",
            ),
            ("      - &unsupported {name: Inline}\n", "Unsupported step anchor or alias"),
        ):
            with self.subTest(steps=steps), self.assertRaisesRegex(AssertionError, diagnostic):
                get_jobs("name: Example\n\njobs:\n  check:\n    steps:\n" + steps)
