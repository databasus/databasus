package members

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultMemberPageLimit = 20
	maxMemberPageLimit     = 100
)

type MemberListQuery struct {
	WorkspaceID uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type MemberPage struct {
	Members []ListedMember
	// NextCursor is nil on the last page.
	NextCursor *string
}

// memberCursor is the position after the last member of a page. The list is ordered by the time a
// member joined, the oldest first, and the ID orders members who joined at the same moment.
type memberCursor struct {
	createdAt time.Time
	memberID  uuid.UUID
}

func (c memberCursor) encode() string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(c.createdAt.UTC().Format(time.RFC3339Nano) + "|" + c.memberID.String()),
	)
}

func decodeMemberCursor(encoded string) (memberCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return memberCursor{}, ErrInvalidCursor
	}

	createdAtText, memberIDText, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return memberCursor{}, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return memberCursor{}, ErrInvalidCursor
	}

	memberID, err := uuid.Parse(memberIDText)
	if err != nil {
		return memberCursor{}, ErrInvalidCursor
	}

	return memberCursor{createdAt: createdAt, memberID: memberID}, nil
}
