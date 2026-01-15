package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.MongodbVersion `json:"version" gorm:"type:text;not null"`

	Host         string `json:"host"         gorm:"type:text;not null"`
	Port         int    `json:"port"         gorm:"type:int;not null"`
	Username     string `json:"username"     gorm:"type:text;not null"`
	Password     string `json:"password"     gorm:"type:text;not null"`
	Database     string `json:"database"     gorm:"type:text;not null"`
	AuthDatabase string `json:"authDatabase" gorm:"type:text;not null;default:'admin'"`
	IsHttps      bool   `json:"isHttps"      gorm:"type:boolean;default:false"`
	CpuCount     int    `json:"cpuCount"     gorm:"column:cpu_count;type:int;not null;default:1"`

	TlsCaFile      string `json:"tlsCaFile"      gorm:"type:text;column:tls_ca_file"`
	TlsCertFile    string `json:"tlsCertFile"    gorm:"type:text;column:tls_cert_file"`
	TlsCertKeyFile string `json:"tlsCertKeyFile" gorm:"type:text;column:tls_cert_key_file"`
}

func (m *MongodbDatabase) TableName() string {
	return "mongodb_databases"
}

func (m *MongodbDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}
	if m.Port == 0 {
		return errors.New("port is required")
	}
	if m.Username == "" {
		return errors.New("username is required")
	}
	if m.Password == "" {
		return errors.New("password is required")
	}
	if m.Database == "" {
		return errors.New("database is required")
	}
	if m.CpuCount <= 0 {
		return errors.New("cpu count must be greater than 0")
	}
	return nil
}

func (m *MongodbDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.buildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	
	// Configure TLS if enabled and certificates are provided
	if m.IsHttps && (m.TlsCaFile != "" || m.TlsCertFile != "" || m.TlsCertKeyFile != "") {
		tlsConfig, err := m.buildTLSConfig(encryptor, databaseID)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("Failed to disconnect from MongoDB", "error", disconnectErr)
		}
	}()

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB database '%s': %w", m.Database, err)
	}

	detectedVersion, err := detectMongodbVersion(ctx, client)
	if err != nil {
		return err
	}
	m.Version = detectedVersion

	if err := checkBackupPermissions(
		ctx,
		client,
		m.Username,
		m.Database,
		m.AuthDatabase,
	); err != nil {
		return err
	}

	return nil
}

func (m *MongodbDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
	m.TlsCaFile = ""
	m.TlsCertFile = ""
	m.TlsCertKeyFile = ""
}

func (m *MongodbDatabase) Update(incoming *MongodbDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.AuthDatabase = incoming.AuthDatabase
	m.IsHttps = incoming.IsHttps
	m.CpuCount = incoming.CpuCount

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
	if incoming.TlsCaFile != "" {
		m.TlsCaFile = incoming.TlsCaFile
	}
	if incoming.TlsCertFile != "" {
		m.TlsCertFile = incoming.TlsCertFile
	}
	if incoming.TlsCertKeyFile != "" {
		m.TlsCertKeyFile = incoming.TlsCertKeyFile
	}
}

func (m *MongodbDatabase) EncryptSensitiveFields(
	databaseID uuid.UUID,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(databaseID, m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}
	if m.TlsCaFile != "" {
		encrypted, err := encryptor.Encrypt(databaseID, m.TlsCaFile)
		if err != nil {
			return err
		}
		m.TlsCaFile = encrypted
	}
	if m.TlsCertFile != "" {
		encrypted, err := encryptor.Encrypt(databaseID, m.TlsCertFile)
		if err != nil {
			return err
		}
		m.TlsCertFile = encrypted
	}
	if m.TlsCertKeyFile != "" {
		encrypted, err := encryptor.Encrypt(databaseID, m.TlsCertKeyFile)
		if err != nil {
			return err
		}
		m.TlsCertKeyFile = encrypted
	}
	return nil
}

func (m *MongodbDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	return m.PopulateVersion(logger, encryptor, databaseID)
}

func (m *MongodbDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.buildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	
	// Configure TLS if enabled and certificates are provided
	if m.IsHttps && (m.TlsCaFile != "" || m.TlsCertFile != "" || m.TlsCertKeyFile != "") {
		tlsConfig, err := m.buildTLSConfig(encryptor, databaseID)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("Failed to disconnect", "error", disconnectErr)
		}
	}()

	detectedVersion, err := detectMongodbVersion(ctx, client)
	if err != nil {
		return err
	}

	m.Version = detectedVersion
	return nil
}

func (m *MongodbDatabase) IsUserReadOnly(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (bool, []string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.buildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	
	// Configure TLS if enabled and certificates are provided
	if m.IsHttps && (m.TlsCaFile != "" || m.TlsCertFile != "" || m.TlsCertKeyFile != "") {
		tlsConfig, err := m.buildTLSConfig(encryptor, databaseID)
		if err != nil {
			return false, nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("Failed to disconnect", "error", disconnectErr)
		}
	}()

	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	adminDB := client.Database(authDB)
	var result bson.M
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: m.Username},
			{Key: "db", Value: authDB},
		}},
	}).Decode(&result)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	writeRoles := map[string]bool{
		"readWrite":            true,
		"readWriteAnyDatabase": true,
		"dbAdmin":              true,
		"dbAdminAnyDatabase":   true,
		"userAdmin":            true,
		"userAdminAnyDatabase": true,
		"clusterAdmin":         true,
		"clusterManager":       true,
		"hostManager":          true,
		"root":                 true,
		"dbOwner":              true,
		"restore":              true,
		"__system":             true,
	}

	// Roles that are read-only for our backup purposes
	// The "backup" role has insert/update on mms.backup collection but is needed for mongodump
	readOnlyRoles := map[string]bool{
		"read":   true,
		"backup": true,
	}

	writeActions := map[string]bool{
		"insert":             true,
		"update":             true,
		"remove":             true,
		"createCollection":   true,
		"dropCollection":     true,
		"createIndex":        true,
		"dropIndex":          true,
		"convertToCapped":    true,
		"dropDatabase":       true,
		"renameCollection":   true,
		"createUser":         true,
		"dropUser":           true,
		"updateUser":         true,
		"grantRole":          true,
		"revokeRole":         true,
		"dropRole":           true,
		"createRole":         true,
		"updateRole":         true,
		"enableSharding":     true,
		"shardCollection":    true,
		"addShard":           true,
		"removeShard":        true,
		"shutdown":           true,
		"replSetReconfig":    true,
		"replSetStateChange": true,
	}

	var detectedRoles []string

	users, ok := result["users"].(bson.A)
	if !ok || len(users) == 0 {
		return true, detectedRoles, nil
	}

	user, ok := users[0].(bson.M)
	if !ok {
		return true, detectedRoles, nil
	}

	roles, ok := user["roles"].(bson.A)
	if !ok {
		return true, detectedRoles, nil
	}

	// Collect all role names and check for write roles
	for _, roleDoc := range roles {
		role, ok := roleDoc.(bson.M)
		if !ok {
			continue
		}
		roleName, _ := role["role"].(string)
		if roleName != "" {
			detectedRoles = append(detectedRoles, roleName)
		}
	}

	// Check if any detected role is a write role
	for _, roleName := range detectedRoles {
		if writeRoles[roleName] {
			return false, detectedRoles, nil
		}
	}

	// If all roles are known read-only roles (read, backup), skip inherited privilege check
	allRolesReadOnly := true
	for _, roleName := range detectedRoles {
		if !readOnlyRoles[roleName] {
			allRolesReadOnly = false
			break
		}
	}
	if allRolesReadOnly && len(detectedRoles) > 0 {
		return true, detectedRoles, nil
	}

	// Check inherited privileges for custom roles
	var privResult bson.M
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: m.Username},
			{Key: "db", Value: authDB},
		}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&privResult)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user privileges: %w", err)
	}

	privUsers, ok := privResult["users"].(bson.A)
	if !ok || len(privUsers) == 0 {
		return true, detectedRoles, nil
	}

	privUser, ok := privUsers[0].(bson.M)
	if !ok {
		return true, detectedRoles, nil
	}

	// Check inheritedPrivileges for write actions
	inheritedPrivileges, ok := privUser["inheritedPrivileges"].(bson.A)
	if ok {
		for _, privDoc := range inheritedPrivileges {
			priv, ok := privDoc.(bson.M)
			if !ok {
				continue
			}
			actions, ok := priv["actions"].(bson.A)
			if !ok {
				continue
			}
			for _, action := range actions {
				actionStr, ok := action.(string)
				if ok && writeActions[actionStr] {
					return false, detectedRoles, nil
				}
			}
		}
	}

	return true, detectedRoles, nil
}

func (m *MongodbDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, string, error) {
	password, err := decryptPasswordIfNeeded(m.Password, encryptor, databaseID)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	uri := m.buildConnectionURI(password)

	clientOptions := options.Client().ApplyURI(uri)
	
	// Configure TLS if enabled and certificates are provided
	if m.IsHttps && (m.TlsCaFile != "" || m.TlsCertFile != "" || m.TlsCertKeyFile != "") {
		tlsConfig, err := m.buildTLSConfig(encryptor, databaseID)
		if err != nil {
			return "", "", fmt.Errorf("failed to configure TLS: %w", err)
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			logger.Error("Failed to disconnect", "error", disconnectErr)
		}
	}()

	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	maxRetries := 3
	for attempt := range maxRetries {
		newUsername := fmt.Sprintf("databasus-%s", uuid.New().String()[:8])
		newPassword := encryption.GenerateComplexPassword()

		adminDB := client.Database(authDB)
		err = adminDB.RunCommand(ctx, bson.D{
			{Key: "createUser", Value: newUsername},
			{Key: "pwd", Value: newPassword},
			{Key: "roles", Value: bson.A{
				bson.D{
					{Key: "role", Value: "backup"},
					{Key: "db", Value: "admin"},
				},
				bson.D{
					{Key: "role", Value: "read"},
					{Key: "db", Value: m.Database},
				},
			}},
		}).Err()
		if err != nil {
			if attempt < maxRetries-1 {
				continue
			}
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		logger.Info(
			"Read-only MongoDB user created successfully",
			"username", newUsername,
		)
		return newUsername, newPassword, nil
	}

	return "", "", errors.New("failed to generate unique username after 3 attempts")
}

// buildTLSConfig creates a TLS configuration for the MongoDB Go driver
func (m *MongodbDatabase) buildTLSConfig(
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load CA certificate for server verification
	if m.TlsCaFile != "" {
		decryptedCA, err := encryptor.Decrypt(databaseID, m.TlsCaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(decryptedCA)) {
			return nil, errors.New("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key for mutual TLS
	if m.TlsCertFile != "" && m.TlsCertKeyFile != "" {
		decryptedCert, err := encryptor.Decrypt(databaseID, m.TlsCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt client certificate: %w", err)
		}

		decryptedKey, err := encryptor.Decrypt(databaseID, m.TlsCertKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt client key: %w", err)
		}

		// Parse the certificate and key
		cert, err := tls.X509KeyPair([]byte(decryptedCert), []byte(decryptedKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate and key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	} else if m.TlsCertFile != "" || m.TlsCertKeyFile != "" {
		return nil, errors.New("both client certificate and key must be provided together")
	}

	return tlsConfig, nil
}

// buildConnectionURI builds a MongoDB connection URI
func (m *MongodbDatabase) buildConnectionURI(password string) string {
	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	tlsParams := ""
	if m.IsHttps {
		// Use tlsInsecure only if no certificates are provided
		if m.TlsCaFile == "" && m.TlsCertFile == "" && m.TlsCertKeyFile == "" {
			tlsParams = "&tls=true&tlsInsecure=true"
		} else {
			tlsParams = "&tls=true"
		}
	}

	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&connectTimeoutMS=15000%s",
		url.QueryEscape(m.Username),
		url.QueryEscape(password),
		m.Host,
		m.Port,
		m.Database,
		authDB,
		tlsParams,
	)
}

// BuildMongodumpURI builds a URI suitable for mongodump (without database in path)
func (m *MongodbDatabase) BuildMongodumpURI(password string) string {
	authDB := m.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	tlsParams := ""
	if m.IsHttps {
		// Use tlsInsecure only if no certificates are provided
		if m.TlsCaFile == "" && m.TlsCertFile == "" && m.TlsCertKeyFile == "" {
			tlsParams = "&tls=true&tlsInsecure=true"
		} else {
			tlsParams = "&tls=true"
		}
	}

	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/?authSource=%s&connectTimeoutMS=15000%s",
		url.QueryEscape(m.Username),
		url.QueryEscape(password),
		m.Host,
		m.Port,
		authDB,
		tlsParams,
	)
}

// detectMongodbVersion gets MongoDB server version from buildInfo command
func detectMongodbVersion(ctx context.Context, client *mongo.Client) (tools.MongodbVersion, error) {
	adminDB := client.Database("admin")
	var result bson.M
	err := adminDB.RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("failed to get MongoDB version: %w", err)
	}

	versionStr, ok := result["version"].(string)
	if !ok {
		return "", errors.New("could not parse MongoDB version from buildInfo")
	}

	re := regexp.MustCompile(`^(\d+)\.`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse MongoDB version: %s", versionStr)
	}

	major := matches[1]

	switch major {
	case "4":
		return tools.MongodbVersion4, nil
	case "5":
		return tools.MongodbVersion5, nil
	case "6":
		return tools.MongodbVersion6, nil
	case "7":
		return tools.MongodbVersion7, nil
	case "8":
		return tools.MongodbVersion8, nil
	default:
		return "", fmt.Errorf(
			"unsupported MongoDB major version: %s (supported: 4.x, 5.x, 6.x, 7.x, 8.x)",
			major,
		)
	}
}

// checkBackupPermissions verifies the user has sufficient privileges for mongodump backup.
// Required: 'read' role on target database OR 'backup' role on admin OR 'readAnyDatabase' role.
func checkBackupPermissions(
	ctx context.Context,
	client *mongo.Client,
	username, database, authDatabase string,
) error {
	authDB := authDatabase
	if authDB == "" {
		authDB = "admin"
	}

	adminDB := client.Database(authDB)
	var result bson.M
	err := adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: username},
			{Key: "db", Value: authDB},
		}},
		{Key: "showPrivileges", Value: true},
	}).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	users, ok := result["users"].(bson.A)
	if !ok || len(users) == 0 {
		return errors.New("insufficient permissions for backup. User not found")
	}

	user, ok := users[0].(bson.M)
	if !ok {
		return errors.New("insufficient permissions for backup. Could not parse user info")
	}

	// Check roles for backup permissions
	roles, ok := user["roles"].(bson.A)
	if !ok {
		return errors.New("insufficient permissions for backup. No roles assigned")
	}

	backupRoles := map[string]bool{
		"backup":               true,
		"root":                 true,
		"readAnyDatabase":      true,
		"dbOwner":              true,
		"__system":             true,
		"clusterAdmin":         true,
		"readWriteAnyDatabase": true,
	}

	var userRoles []string
	hasBackupRole := false
	hasReadOnTargetDB := false

	for _, roleDoc := range roles {
		role, ok := roleDoc.(bson.M)
		if !ok {
			continue
		}
		roleName, _ := role["role"].(string)
		roleDB, _ := role["db"].(string)

		if roleName != "" {
			userRoles = append(userRoles, roleName)
		}

		if backupRoles[roleName] {
			hasBackupRole = true
		}

		if roleName == "read" && (roleDB == database || roleDB == "") {
			hasReadOnTargetDB = true
		}
		if roleName == "readWrite" && (roleDB == database || roleDB == "") {
			hasReadOnTargetDB = true
		}
	}

	if hasBackupRole || hasReadOnTargetDB {
		return nil
	}

	// Check inherited privileges for 'find' action on target database
	inheritedPrivileges, ok := user["inheritedPrivileges"].(bson.A)
	if ok {
		for _, privDoc := range inheritedPrivileges {
			priv, ok := privDoc.(bson.M)
			if !ok {
				continue
			}
			resource, ok := priv["resource"].(bson.M)
			if !ok {
				continue
			}

			resourceDB, _ := resource["db"].(string)
			resourceCluster, _ := resource["cluster"].(bool)

			isTargetDB := resourceDB == database || resourceDB == "" || resourceCluster

			actions, ok := priv["actions"].(bson.A)
			if !ok {
				continue
			}

			for _, action := range actions {
				actionStr, ok := action.(string)
				if ok && actionStr == "find" && isTargetDB {
					return nil
				}
			}
		}
	}

	return fmt.Errorf(
		"insufficient permissions for backup. Current roles: %s. Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
		strings.Join(userRoles, ", "),
		database,
	)
}

func decryptPasswordIfNeeded(
	password string,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (string, error) {
	if encryptor == nil {
		return password, nil
	}
	return encryptor.Decrypt(databaseID, password)
}

// TlsCertPaths holds paths to temporary certificate files
type TlsCertPaths struct {
	CaFile         string
	CertKeyFile    string
}

// WriteTempCertificates writes decrypted certificates to temporary files
// Returns paths to the temp files and cleanup function
// MongoDB requires certificate and key in a single PEM file for --sslPEMKeyFile
func (m *MongodbDatabase) WriteTempCertificates(
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) (*TlsCertPaths, func(), error) {
	var paths TlsCertPaths
	var createdFiles []string

	cleanup := func() {
		for _, file := range createdFiles {
			_ = os.Remove(file)
		}
	}

	// Write CA certificate file (for server verification)
	if m.TlsCaFile != "" {
		decrypted, err := encryptor.Decrypt(databaseID, m.TlsCaFile)
		if err != nil {
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to decrypt CA certificate: %w", err)
		}

		tmpFile, err := os.CreateTemp("", "mongodb-ca-*.pem")
		if err != nil {
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to create temp CA file: %w", err)
		}

		if _, err := tmpFile.WriteString(decrypted); err != nil {
			_ = tmpFile.Close()
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to write CA certificate: %w", err)
		}

		if err := tmpFile.Chmod(0o600); err != nil {
			_ = tmpFile.Close()
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to set CA file permissions: %w", err)
		}

		_ = tmpFile.Close()
		paths.CaFile = tmpFile.Name()
		createdFiles = append(createdFiles, paths.CaFile)
	}

	// Write combined certificate+key file (for client authentication)
	// MongoDB's --sslPEMKeyFile expects both certificate and key in a single file
	if m.TlsCertFile != "" && m.TlsCertKeyFile != "" {
		decryptedCert, err := encryptor.Decrypt(databaseID, m.TlsCertFile)
		if err != nil {
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to decrypt client certificate: %w", err)
		}

		decryptedKey, err := encryptor.Decrypt(databaseID, m.TlsCertKeyFile)
		if err != nil {
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to decrypt certificate key: %w", err)
		}

		tmpFile, err := os.CreateTemp("", "mongodb-client-*.pem")
		if err != nil {
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to create temp client cert file: %w", err)
		}

		// Write certificate first, then key (standard PEM bundle format)
		combined := decryptedCert
		if !strings.HasSuffix(combined, "\n") {
			combined += "\n"
		}
		combined += decryptedKey

		if _, err := tmpFile.WriteString(combined); err != nil {
			_ = tmpFile.Close()
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to write client certificate+key: %w", err)
		}

		if err := tmpFile.Chmod(0o600); err != nil {
			_ = tmpFile.Close()
			cleanup()
			return nil, cleanup, fmt.Errorf("failed to set client cert file permissions: %w", err)
		}

		_ = tmpFile.Close()
		paths.CertKeyFile = tmpFile.Name()
		createdFiles = append(createdFiles, paths.CertKeyFile)
	} else if m.TlsCertFile != "" || m.TlsCertKeyFile != "" {
		cleanup()
		return nil, cleanup, errors.New("both client certificate and key must be provided together")
	}

	return &paths, cleanup, nil
}
