package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Source is the engine's handle on one MongoDB database.
type Source struct {
	settings Settings
	// generateUsername is a seam: a test makes the generated name collide with an existing user.
	generateUsername func() string
}

func NewSource(settings Settings) *Source {
	return &Source{settings: settings, generateUsername: generateReadOnlyUsername}
}

// GetSettings returns the settings the source connects with. Through a tunnel they carry the local
// end of the tunnel, so a client tool the caller starts reaches the same server.
func (s *Source) GetSettings() Settings {
	return s.settings
}

// TestConnection connects, detects the version and checks that the user can read every collection
// mongodump would dump, leaving out excludeCollections.
func (s *Source) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	excludeCollections []string,
) (Version, error) {
	client, err := s.connect()
	if err != nil {
		// The URI carries the password, so only the database name goes into the log.
		logger.ErrorContext(ctx, "failed to open the mongodb connection",
			slog.String("database_name", s.settings.Database),
			slog.Any("error", err),
		)

		return "", fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer disconnect(ctx, client, logger)

	if err := client.Ping(ctx, nil); err != nil {
		logger.ErrorContext(ctx, "failed to ping the mongodb database",
			slog.String("database_name", s.settings.Database),
			slog.Any("error", err),
		)

		return "", fmt.Errorf("failed to ping MongoDB database '%s': %w", s.settings.Database, err)
	}

	version, err := detectVersion(ctx, client)
	if err != nil {
		return "", err
	}

	if err := checkDumpReadPrivileges(ctx, client, dumpScope{
		Username:           s.settings.Username,
		Database:           s.settings.Database,
		AuthDatabase:       s.settings.getAuthDatabaseName(),
		ExcludeCollections: excludeCollections,
	}); err != nil {
		return "", err
	}

	return version, nil
}

func (s *Source) DetectVersion(ctx context.Context, logger *slog.Logger) (Version, error) {
	client, err := s.connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer disconnect(ctx, client, logger)

	return detectVersion(ctx, client)
}

// GetRawSizeMb reads dataSize from the dbStats command: the uncompressed total of user data, the
// closest analog to the relational "raw DB size". Without a database name there is nothing to
// measure, so it returns 0.
func (s *Source) GetRawSizeMb(ctx context.Context, logger *slog.Logger) (float64, error) {
	if s.settings.Database == "" {
		return 0, nil
	}

	client, err := s.connect()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer disconnect(ctx, client, logger)

	var stats struct {
		DataSize float64 `bson:"dataSize"`
	}

	if err := client.Database(s.settings.Database).
		RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).
		Decode(&stats); err != nil {
		return 0, fmt.Errorf("failed to decode dbStats for MongoDB database '%s': %w", s.settings.Database, err)
	}

	return stats.DataSize / (1024 * 1024), nil
}

func (s *Source) connect() (*mongo.Client, error) {
	return mongo.Connect(options.Client().
		ApplyURI(s.settings.BuildConnectionURI()).
		SetBSONOptions(&options.BSONOptions{DefaultDocumentM: true}))
}

func disconnect(ctx context.Context, client *mongo.Client, logger *slog.Logger) {
	if err := client.Disconnect(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to disconnect from mongodb", slog.Any("error", err))
	}
}

func generateReadOnlyUsername() string {
	return "databasus-" + uuid.New().String()[:8]
}
