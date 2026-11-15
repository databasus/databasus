package configs

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

// DestinationChoice is a storage root admin can select, without its settings.
type DestinationChoice struct {
	StorageID     uuid.UUID
	StorageName   string
	StorageType   storages.Type
	WorkspaceID   uuid.UUID
	WorkspaceName string
	IsSelected    bool
}

type DestinationChoiceQuery struct {
	Limit int
	After *string
	Query *string
}

type DestinationChoicePage struct {
	Items      []DestinationChoice
	NextCursor *string
}

// choiceCursor is the position after the last choice of a page. Choices are ordered by workspace
// name, storage name and storage ID across modules, so the page is assembled in Go.
type choiceCursor struct {
	WorkspaceName string    `json:"w"`
	StorageName   string    `json:"s"`
	StorageID     uuid.UUID `json:"i"`
}

// ListDestinationChoices pages every storage of every workspace. An instance holds few enough
// storages that reading them all is cheaper than a cross-module join.
func (s *Service) ListDestinationChoices(
	ctx context.Context,
	actor users.Actor,
	query DestinationChoiceQuery,
) (DestinationChoicePage, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return DestinationChoicePage{}, err
	}

	var after *choiceCursor
	if query.After != nil {
		decoded, err := decodeChoiceCursor(*query.After)
		if err != nil {
			return DestinationChoicePage{}, newInvalidCursorProblem()
		}

		after = &decoded
	}

	choices, err := s.loadChoices(ctx)
	if err != nil {
		return DestinationChoicePage{}, err
	}

	choices = filterChoices(choices, query.Query)
	slices.SortFunc(choices, compareChoices)

	return pageChoices(choices, after, query.Limit), nil
}

func (s *Service) loadChoices(ctx context.Context) ([]DestinationChoice, error) {
	allStorages, err := s.storages.GetAllStorages(ctx)
	if err != nil {
		return nil, err
	}

	workspaceIDs := make([]uuid.UUID, 0, len(allStorages))
	for _, storage := range allStorages {
		workspaceIDs = append(workspaceIDs, storage.WorkspaceID)
	}

	foundWorkspaces, err := s.workspaces.GetWorkspaces(ctx, uniqueIDs(workspaceIDs))
	if err != nil {
		return nil, err
	}

	workspaceNames := make(map[uuid.UUID]string, len(foundWorkspaces))
	for _, workspace := range foundWorkspaces {
		workspaceNames[workspace.ID] = workspace.Name
	}

	destinations, err := s.destinations.list(ctx)
	if err != nil {
		return nil, err
	}

	selectedIDs := make(map[uuid.UUID]bool, len(destinations))
	for _, destination := range destinations {
		selectedIDs[destination.StorageID] = true
	}

	choices := make([]DestinationChoice, 0, len(allStorages))
	for _, storage := range allStorages {
		choices = append(choices, DestinationChoice{
			StorageID:     storage.ID,
			StorageName:   storage.Name,
			StorageType:   storage.Type,
			WorkspaceID:   storage.WorkspaceID,
			WorkspaceName: workspaceNames[storage.WorkspaceID],
			IsSelected:    selectedIDs[storage.ID],
		})
	}

	return choices, nil
}

// GetDestinationSummaries describes the current destinations with the current storage and workspace
// names, in storage ID order.
func (s *Service) GetDestinationSummaries(ctx context.Context) ([]DestinationChoice, error) {
	choices, err := s.loadChoices(ctx)
	if err != nil {
		return nil, err
	}

	selectedChoices := []DestinationChoice{}
	for _, choice := range choices {
		if choice.IsSelected {
			selectedChoices = append(selectedChoices, choice)
		}
	}

	return selectedChoices, nil
}

func filterChoices(choices []DestinationChoice, query *string) []DestinationChoice {
	if query == nil || *query == "" {
		return choices
	}

	needle := strings.ToLower(*query)

	return slices.DeleteFunc(choices, func(choice DestinationChoice) bool {
		return !strings.Contains(strings.ToLower(choice.StorageName), needle) &&
			!strings.Contains(strings.ToLower(choice.WorkspaceName), needle)
	})
}

func compareChoices(left, right DestinationChoice) int {
	return cmp.Or(
		strings.Compare(left.WorkspaceName, right.WorkspaceName),
		strings.Compare(left.StorageName, right.StorageName),
		strings.Compare(left.StorageID.String(), right.StorageID.String()),
	)
}

func pageChoices(choices []DestinationChoice, after *choiceCursor, limit int) DestinationChoicePage {
	start := 0
	if after != nil {
		position := DestinationChoice{
			StorageID:     after.StorageID,
			StorageName:   after.StorageName,
			WorkspaceName: after.WorkspaceName,
			StorageType:   "",
			WorkspaceID:   uuid.Nil,
			IsSelected:    false,
		}
		start, _ = slices.BinarySearchFunc(choices, position, compareChoices)

		if start < len(choices) && compareChoices(choices[start], position) == 0 {
			start++
		}
	}

	end := min(start+limit, len(choices))
	page := DestinationChoicePage{Items: choices[start:end], NextCursor: nil}

	if end < len(choices) {
		last := choices[end-1]
		cursor := encodeChoiceCursor(choiceCursor{
			WorkspaceName: last.WorkspaceName,
			StorageName:   last.StorageName,
			StorageID:     last.StorageID,
		})
		page.NextCursor = &cursor
	}

	return page
}

func encodeChoiceCursor(cursor choiceCursor) string {
	encoded, _ := json.Marshal(cursor)

	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeChoiceCursor(encoded string) (choiceCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return choiceCursor{}, err
	}

	var cursor choiceCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return choiceCursor{}, err
	}

	return cursor, nil
}

func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	sortedIDs := slices.Clone(ids)
	slices.SortFunc(sortedIDs, compareIDs)

	return slices.Compact(sortedIDs)
}
