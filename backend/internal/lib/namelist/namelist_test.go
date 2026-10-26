package namelist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUniqueNames(t *testing.T) {
	t.Parallel()

	t.Run("returns an empty slice with blank input", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, ParseUniqueNames(""))
		assert.Empty(t, ParseUniqueNames("   "))
		assert.Empty(t, ParseUniqueNames(",,,"))
		assert.NotNil(t, ParseUniqueNames(""))
	})

	t.Run("returns trimmed names with pasted multiline input", func(t *testing.T) {
		t.Parallel()
		pastedNames := "personnel_access_control_event,\npersonnel_real_time_location,\r\n personnel_real_time\t"

		assert.Equal(
			t,
			[]string{"personnel_access_control_event", "personnel_real_time_location", "personnel_real_time"},
			ParseUniqueNames(pastedNames),
		)
	})

	t.Run("returns unique names with duplicates and empty entries", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []string{"orders", "users"}, ParseUniqueNames("orders,, users ,orders"))
	})
}

func TestNormalizeUniqueNames(t *testing.T) {
	t.Parallel()

	t.Run("splits entries apart with separators inside them", func(t *testing.T) {
		t.Parallel()
		storedNames := []string{"orders", "\nusers, payments", "   ", "orders"}

		assert.Equal(t, []string{"orders", "users", "payments"}, NormalizeUniqueNames(storedNames))
	})
}

func TestFormatUniqueNames(t *testing.T) {
	t.Parallel()

	t.Run("joins normalized names with commas with dirty names", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "orders,users", FormatUniqueNames([]string{" orders ", "\nusers", "orders", ""}))
		assert.Empty(t, FormatUniqueNames(nil))
	})
}

func TestFormatTruncatedNames(t *testing.T) {
	t.Parallel()

	t.Run("lists every name when the names fit the limit", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "orders, users", FormatTruncatedNames([]string{"orders", "users"}))
		assert.Empty(t, FormatTruncatedNames(nil))
	})

	t.Run("appends the remaining count when the names exceed the limit", func(t *testing.T) {
		t.Parallel()
		manyNames := []string{"a", "b", "c", "d", "e", "f", "g"}

		assert.Equal(t, "a, b, c, d, e and 2 more", FormatTruncatedNames(manyNames))
	})
}
