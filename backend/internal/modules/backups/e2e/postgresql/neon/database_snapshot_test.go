package neon

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

// seedStatements give the source what a restore can lose: rows in several tables, a foreign key,
// sequences moved past their start, a function and a view that calls it.
const seedStatements = `
CREATE TABLE public.customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES public.customers (id),
    amount NUMERIC(10, 2) NOT NULL
);
CREATE SEQUENCE public.invoice_numbers START 1000;
INSERT INTO public.customers (name, joined_at) VALUES
    ('Alice', '2026-01-05 10:00:00+00'),
    ('Bob', '2026-02-11 12:30:00+00'),
    ('Carol', '2026-03-20 08:15:00+00');
INSERT INTO public.orders (customer_id, amount) VALUES (1, 19.99), (1, 5.00), (2, 42.50);
SELECT setval('public.invoice_numbers', 1042);
CREATE FUNCTION public.customer_total(target_customer_id INTEGER) RETURNS NUMERIC
    LANGUAGE sql STABLE
    AS $$ SELECT COALESCE(SUM(amount), 0) FROM public.orders WHERE customer_id = target_customer_id $$;
CREATE VIEW public.customer_totals AS
    SELECT id, name, public.customer_total(id) AS total FROM public.customers;
`

// seededValueCount is what a snapshot of the seeded source holds: three rows of each table, three
// view rows and three sequences.
const seededValueCount = 12

// getSnapshotQueries read each compared value as a name and a text, so a difference names what
// differs.
func getSnapshotQueries() []string {
	return []string{
		`SELECT 'table customers row ' || id, row_to_json(customers)::text FROM public.customers ORDER BY id`,
		`SELECT 'table orders row ' || id, row_to_json(orders)::text FROM public.orders ORDER BY id`,
		`SELECT 'view customer_totals row ' || id, row_to_json(customer_totals)::text
		FROM public.customer_totals ORDER BY id`,
		`SELECT 'sequence ' || sequencename, COALESCE(last_value::text, 'not called')
		FROM pg_sequences WHERE schemaname = 'public' ORDER BY sequencename`,
	}
}

// databaseSnapshot holds every compared value of one database by its name.
type databaseSnapshot struct {
	databaseName string
	values       map[string]string
}

func takeSnapshot(reporter testReporter, server managedServer, databaseName string) databaseSnapshot {
	reporter.Helper()

	conn := server.open(reporter, databaseName)
	snapshot := databaseSnapshot{databaseName: databaseName, values: map[string]string{}}

	for _, query := range getSnapshotQueries() {
		rows, err := conn.Query(reporter.Context(), query)
		if err != nil {
			reporter.Fatalf("read %s: %s", server.describe(databaseName), server.redactor.redact(err.Error()))
		}

		namedValues, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) ([2]string, error) {
			var namedValue [2]string
			err := row.Scan(&namedValue[0], &namedValue[1])

			return namedValue, err
		})
		if err != nil {
			reporter.Fatalf("read %s: %s", server.describe(databaseName), server.redactor.redact(err.Error()))
		}

		for _, namedValue := range namedValues {
			snapshot.values[namedValue[0]] = namedValue[1]
		}
	}

	return snapshot
}

// findSnapshotDifferences names every value the restored database misses, holds differently or
// holds beyond the source.
func findSnapshotDifferences(source, restored databaseSnapshot) []string {
	var differences []string

	for _, valueName := range slices.Sorted(maps.Keys(source.values)) {
		restoredValue, isRestored := restored.values[valueName]

		switch {
		case !isRestored:
			differences = append(differences, fmt.Sprintf("%s is missing from %s", valueName, restored.databaseName))
		case restoredValue != source.values[valueName]:
			differences = append(differences, fmt.Sprintf("%s is %s in %s but %s in the source %s",
				valueName, restoredValue, restored.databaseName, source.values[valueName], source.databaseName))
		}
	}

	for _, valueName := range slices.Sorted(maps.Keys(restored.values)) {
		if _, isInSource := source.values[valueName]; !isInSource {
			differences = append(differences, fmt.Sprintf("%s is in %s but not in the source %s",
				valueName, restored.databaseName, source.databaseName))
		}
	}

	return differences
}

func TestFindSnapshotDifferences(t *testing.T) {
	t.Parallel()

	sourceSnapshot := databaseSnapshot{databaseName: "source", values: map[string]string{
		"table customers row 1":      `{"id":1,"name":"Alice"}`,
		"table orders row 3":         `{"id":3,"amount":42.50}`,
		"view customer_totals row 1": `{"id":1,"total":24.99}`,
		"sequence invoice_numbers":   "1042",
	}}

	t.Run("finds no difference when every value matches", func(t *testing.T) {
		t.Parallel()
		restoredSnapshot := databaseSnapshot{databaseName: "restored", values: maps.Clone(sourceSnapshot.values)}

		assert.Empty(t, findSnapshotDifferences(sourceSnapshot, restoredSnapshot))
	})

	t.Run("names each differing row, view result, sequence position and extra value", func(t *testing.T) {
		t.Parallel()
		restoredSnapshot := databaseSnapshot{databaseName: "restored", values: map[string]string{
			"table customers row 1":      `{"id":1,"name":"Alicia"}`,
			"view customer_totals row 1": `{"id":1,"total":0}`,
			"sequence invoice_numbers":   "1000",
			"table orders row 4":         `{"id":4,"amount":1.00}`,
		}}

		differences := findSnapshotDifferences(sourceSnapshot, restoredSnapshot)

		assert.ElementsMatch(t, []string{
			`table customers row 1 is {"id":1,"name":"Alicia"} in restored but {"id":1,"name":"Alice"} in the source source`,
			`table orders row 3 is missing from restored`,
			`view customer_totals row 1 is {"id":1,"total":0} in restored but {"id":1,"total":24.99} in the source source`,
			`sequence invoice_numbers is 1000 in restored but 1042 in the source source`,
			`table orders row 4 is in restored but not in the source source`,
		}, differences)
	})
}
