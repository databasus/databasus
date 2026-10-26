package databases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	lib_database "databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
)

type DatabaseRepository struct{}

func (r *DatabaseRepository) Save(database *Database) (*Database, error) {
	db := storage.GetDb()

	isNew := database.ID == uuid.Nil
	if isNew {
		database.ID = uuid.New()
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := database.Validate(); err != nil {
			return err
		}

		withoutAssociations := tx.Omit(
			"PostgresqlLogical", "PostgresqlPhysical", "Mysql", "Mariadb", "Mongodb",
		)

		if isNew {
			if err := withoutAssociations.Create(database).Error; err != nil {
				return err
			}
		} else {
			if err := withoutAssociations.Save(database).Error; err != nil {
				return err
			}
		}

		switch database.Type {
		case DatabaseTypePostgresLogical:
			database.PostgresqlLogical.DatabaseID = &database.ID
			if database.PostgresqlLogical.ID == uuid.Nil {
				database.PostgresqlLogical.ID = uuid.New()
				if err := tx.Create(database.PostgresqlLogical).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.PostgresqlLogical).Error; err != nil {
					return err
				}
			}
		case DatabaseTypePostgresPhysical:
			database.PostgresqlPhysical.DatabaseID = &database.ID
			if database.PostgresqlPhysical.ID == uuid.Nil {
				database.PostgresqlPhysical.ID = uuid.New()
				if err := tx.Create(database.PostgresqlPhysical).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.PostgresqlPhysical).Error; err != nil {
					return err
				}
			}
		case DatabaseTypeMysql:
			database.Mysql.DatabaseID = &database.ID
			if database.Mysql.ID == uuid.Nil {
				database.Mysql.ID = uuid.New()
				if err := tx.Create(database.Mysql).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mysql).Error; err != nil {
					return err
				}
			}
		case DatabaseTypeMariadb:
			database.Mariadb.DatabaseID = &database.ID
			if database.Mariadb.ID == uuid.Nil {
				database.Mariadb.ID = uuid.New()
				if err := tx.Create(database.Mariadb).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mariadb).Error; err != nil {
					return err
				}
			}
		case DatabaseTypeMongodb:
			database.Mongodb.DatabaseID = &database.ID
			if database.Mongodb.ID == uuid.Nil {
				database.Mongodb.ID = uuid.New()
				if err := tx.Create(database.Mongodb).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mongodb).Error; err != nil {
					return err
				}
			}
		}

		return replaceNotifierIDs(tx, database.ID, database.GetNotifierIDs())
	})
	if err != nil {
		return nil, err
	}

	return database, nil
}

// UpdateColumns writes only the given columns of one database row, so a background status update
// cannot overwrite settings a user saved after the caller read the database. A map is used so a
// column cleared to NULL is written instead of being skipped as a zero value.
func (r *DatabaseRepository) UpdateColumns(id uuid.UUID, columns map[string]any) error {
	result := storage.GetDb().Model(&Database{}).Where("id = ?", id).Updates(columns)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("database %s not found: %w", id, gorm.ErrRecordNotFound)
	}

	return nil
}

func (r *DatabaseRepository) FindByID(id uuid.UUID) (*Database, error) {
	var database Database

	if err := storage.
		GetDb().
		Preload("PostgresqlLogical").
		Preload("PostgresqlPhysical").
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
		Where("id = ?", id).
		First(&database).Error; err != nil {
		return nil, err
	}

	if err := fillNotifierIDs(storage.GetDb(), []*Database{&database}); err != nil {
		return nil, err
	}

	return &database, nil
}

func (r *DatabaseRepository) FindByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]*Database, error) {
	var databases []*Database

	if err := lib_database.Conn(ctx, storage.GetDb()).
		Preload("PostgresqlLogical").
		Preload("PostgresqlPhysical").
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
		Where("workspace_id = ?", workspaceID).
		Order("CASE WHEN health_status = 'UNAVAILABLE' THEN 1 WHEN health_status = 'AVAILABLE' THEN 2 WHEN health_status IS NULL THEN 3 ELSE 4 END, name ASC").
		Find(&databases).Error; err != nil {
		return nil, err
	}

	if err := fillNotifierIDs(lib_database.Conn(ctx, storage.GetDb()), databases); err != nil {
		return nil, err
	}

	return databases, nil
}

func (r *DatabaseRepository) Delete(id uuid.UUID) error {
	db := storage.GetDb()

	return db.Transaction(func(tx *gorm.DB) error {
		var database Database
		if err := tx.Where("id = ?", id).First(&database).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM database_notifiers WHERE database_id = ?", id).Error; err != nil {
			return err
		}

		switch database.Type {
		case DatabaseTypePostgresLogical:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&postgresql_logical.PostgresqlLogicalDatabase{}).Error; err != nil {
				return err
			}
		case DatabaseTypeMysql:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mysql.MysqlDatabase{}).Error; err != nil {
				return err
			}
		case DatabaseTypeMariadb:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mariadb.MariadbDatabase{}).Error; err != nil {
				return err
			}
		case DatabaseTypeMongodb:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mongodb.MongodbDatabase{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Delete(&Database{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *DatabaseRepository) IsNotifierUsing(notifierID uuid.UUID) (bool, error) {
	var count int64

	if err := storage.
		GetDb().
		Table("database_notifiers").
		Where("notifier_id = ?", notifierID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *DatabaseRepository) GetAllDatabases() ([]*Database, error) {
	var databases []*Database

	if err := storage.
		GetDb().
		Preload("PostgresqlLogical").
		Preload("PostgresqlPhysical").
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
		Find(&databases).Error; err != nil {
		return nil, err
	}

	if err := fillNotifierIDs(storage.GetDb(), databases); err != nil {
		return nil, err
	}

	return databases, nil
}

func (r *DatabaseRepository) GetDatabasesIDsByNotifierID(
	notifierID uuid.UUID,
) ([]uuid.UUID, error) {
	var databasesIDs []uuid.UUID

	if err := storage.
		GetDb().
		Table("database_notifiers").
		Where("notifier_id = ?", notifierID).
		Pluck("database_id", &databasesIDs).Error; err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

// CountDatabasesUsingNotifierOutsideWorkspace counts the databases that use the notifier and are
// not in the workspace. It runs in the transaction ctx carries.
func (r *DatabaseRepository) CountDatabasesUsingNotifierOutsideWorkspace(
	ctx context.Context,
	notifierID uuid.UUID,
	workspaceID uuid.UUID,
) (int64, error) {
	var count int64

	err := lib_database.Conn(ctx, storage.GetDb()).Raw(
		`SELECT COUNT(*)
		FROM database_notifiers dn
		JOIN databases d ON d.id = dn.database_id
		WHERE dn.notifier_id = ? AND d.workspace_id IS DISTINCT FROM ?`,
		notifierID, workspaceID,
	).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count the databases using the notifier outside the workspace: %w", err)
	}

	return count, nil
}

func replaceNotifierIDs(tx *gorm.DB, databaseID uuid.UUID, notifierIDs []uuid.UUID) error {
	if err := tx.Exec("DELETE FROM database_notifiers WHERE database_id = ?", databaseID).Error; err != nil {
		return err
	}

	for _, notifierID := range notifierIDs {
		err := tx.Exec(
			"INSERT INTO database_notifiers (database_id, notifier_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			databaseID, notifierID,
		).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// fillNotifierIDs sets the notifiers of the databases to the IDs they use; the service adds the
// other fields from the notifiers module.
func fillNotifierIDs(db *gorm.DB, databases []*Database) error {
	if len(databases) == 0 {
		return nil
	}

	databaseIDs := make([]uuid.UUID, 0, len(databases))
	for _, database := range databases {
		databaseIDs = append(databaseIDs, database.ID)
	}

	var attachments []struct {
		DatabaseID uuid.UUID
		NotifierID uuid.UUID
	}

	if err := db.Raw(
		"SELECT database_id, notifier_id FROM database_notifiers WHERE database_id IN ? ORDER BY notifier_id",
		databaseIDs,
	).Scan(&attachments).Error; err != nil {
		return err
	}

	notifiersByDatabaseID := make(map[uuid.UUID][]DatabaseNotifier, len(databases))
	for _, attachment := range attachments {
		notifiersByDatabaseID[attachment.DatabaseID] = append(
			notifiersByDatabaseID[attachment.DatabaseID], DatabaseNotifier{ID: attachment.NotifierID},
		)
	}

	for _, database := range databases {
		database.Notifiers = notifiersByDatabaseID[database.ID]
	}

	return nil
}
