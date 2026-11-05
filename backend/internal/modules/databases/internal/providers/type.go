package providers

// Type is the kind of database a provider serves. It is fixed when the database is created.
type Type string

const (
	TypePostgresqlLogical  Type = "POSTGRES_LOGICAL"
	TypePostgresqlPhysical Type = "POSTGRES_PHYSICAL"
	TypeMysql              Type = "MYSQL"
	TypeMariadb            Type = "MARIADB"
	TypeMongodb            Type = "MONGODB"
)
