from dataclasses import dataclass


@dataclass(frozen=True)
class RowComparison:
    written_rows: list[str]
    stored_rows: list[str]

    def describe_difference(self) -> str | None:
        if self.written_rows == self.stored_rows:
            return None

        missing_rows = [row for row in self.written_rows if row not in self.stored_rows]
        unexpected_rows = [row for row in self.stored_rows if row not in self.written_rows]

        return (
            f"missing rows {missing_rows}, unexpected rows {unexpected_rows}, "
            f"{len(self.written_rows)} rows written, {len(self.stored_rows)} stored"
        )
