package databases

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
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
			"PostgresqlLogical", "PostgresqlPhysical", "Mysql", "Mariadb", "Mongodb", "Notifiers",
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

		if err := tx.
			Model(database).
			Association("Notifiers").
			Replace(database.Notifiers); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return database, nil
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
		Preload("Notifiers").
		Where("id = ?", id).
		First(&database).Error; err != nil {
		return nil, err
	}

	return &database, nil
}

func (r *DatabaseRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]*Database, error) {
	var databases []*Database

	if err := storage.
		GetDb().
		Preload("PostgresqlLogical").
		Preload("PostgresqlPhysical").
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
		Preload("Notifiers").
		Where("workspace_id = ?", workspaceID).
		Order("CASE WHEN health_status = 'UNAVAILABLE' THEN 1 WHEN health_status = 'AVAILABLE' THEN 2 WHEN health_status IS NULL THEN 3 ELSE 4 END, name ASC").
		Find(&databases).Error; err != nil {
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

		if err := tx.Model(&database).Association("Notifiers").Clear(); err != nil {
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

func (r *DatabaseRepository) CountAll() (int64, error) {
	var count int64

	if err := storage.GetDb().Model(&Database{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
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
		Preload("Notifiers").
		Find(&databases).Error; err != nil {
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
