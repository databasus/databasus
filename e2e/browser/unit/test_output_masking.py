from output_masking import OutputMask


class TestOutputMask:
    def test_masks_a_generated_password_wherever_it_appears_in_a_line(self) -> None:
        output_mask = OutputMask(["generated-password"])

        masked_line = output_mask.apply(
            "connect password=generated-password failed for generated-password"
        )

        assert "generated-password" not in masked_line

    def test_masks_any_bearer_token_without_knowing_its_value(self) -> None:
        output_mask = OutputMask([])

        masked_line = output_mask.apply("Authorization: Bearer some.session-token next")

        assert "some.session-token" not in masked_line
        assert masked_line.endswith("next")

    def test_leaves_a_line_without_secrets_unchanged(self) -> None:
        output_mask = OutputMask(["generated-password"])

        assert output_mask.apply("nothing to hide") == "nothing to hide"

    def test_ignores_an_empty_secret(self) -> None:
        output_mask = OutputMask([""])

        assert output_mask.apply("text") == "text"

    def test_masks_restore_query_tokens_in_plain_and_encoded_diagnostics(self) -> None:
        from urllib.parse import quote

        url = (
            "http://databasus:4005/api/v1/physical-restore-sets?token=restore-secret&other=visible"
        )
        output_mask = OutputMask([])

        for diagnostic in (url, quote(url, safe=""), quote(quote(url, safe=""), safe="")):
            masked_line = output_mask.apply(diagnostic)

            assert "restore-secret" not in masked_line
            assert "other=visible" in masked_line
