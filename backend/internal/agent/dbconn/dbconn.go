package dbconn

import "fmt"

type Conn struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func (c Conn) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.Database,
	)
}
