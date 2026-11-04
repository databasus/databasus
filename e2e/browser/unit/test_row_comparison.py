from row_comparison import RowComparison


class TestDescribeDifference:
    def test_reports_nothing_when_the_rows_match(self):
        assert RowComparison(["first", "second"], ["first", "second"]).describe_difference() is None

    def test_names_the_missing_row_when_a_restore_loses_one(self):
        difference = RowComparison(["first", "second"], ["first"]).describe_difference()

        assert difference is not None
        assert "missing rows ['second']" in difference

    def test_names_the_unexpected_row_when_a_restore_adds_one(self):
        difference = RowComparison(["first"], ["first", "extra"]).describe_difference()

        assert difference is not None
        assert "unexpected rows ['extra']" in difference
