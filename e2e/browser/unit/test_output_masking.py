from output_masking import OutputMask


class TestOutputMask:
    def test_masks_a_generated_password_wherever_it_appears_in_a_line(self):
        output_mask = OutputMask(["generated-password"])

        masked_line = output_mask.apply("connect password=generated-password failed for generated-password")

        assert "generated-password" not in masked_line

    def test_masks_any_bearer_token_without_knowing_its_value(self):
        output_mask = OutputMask([])

        masked_line = output_mask.apply("Authorization: Bearer some.session-token next")

        assert "some.session-token" not in masked_line
        assert masked_line.endswith("next")

    def test_leaves_a_line_without_secrets_unchanged(self):
        output_mask = OutputMask(["generated-password"])

        assert output_mask.apply("nothing to hide") == "nothing to hide"

    def test_ignores_an_empty_secret(self):
        output_mask = OutputMask([""])

        assert output_mask.apply("text") == "text"
