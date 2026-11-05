package apptest

import (
	"net/http"
	"testing"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// ListAllAuditEvents follows nextCursor from the first page of the view the filters choose to the
// last, and fails the test unless every page answers 200. The filters' Limit sets the page size.
func ListAllAuditEvents(
	tb testing.TB,
	reader Session,
	filters apiclient.ListAuditEventsParams,
) []apiclient.AuditEvent {
	tb.Helper()

	listedEvents := []apiclient.AuditEvent{}

	for {
		response, err := reader.Client.ListAuditEventsWithResponse(tb.Context(), &filters)
		if err != nil {
			tb.Fatalf("list the audit events: %v", err)
		}

		if response.StatusCode() != http.StatusOK {
			tb.Fatalf("list the audit events answered %d: %s", response.StatusCode(), response.Body)
		}

		listedEvents = append(listedEvents, response.JSON200.Items...)

		if response.JSON200.NextCursor == nil {
			return listedEvents
		}

		filters.After = response.JSON200.NextCursor
	}
}
