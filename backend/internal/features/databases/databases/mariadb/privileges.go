package mariadb

import "databasus-backend/internal/engines/mysqlfamily"

func (m *MariadbDatabase) HasPrivilege(privilege string) bool {
	return mysqlfamily.HasPrivilege(m.Privileges, privilege)
}
