package mysql

import "databasus-backend/internal/engines/mysqlfamily"

func (m *MysqlDatabase) HasPrivilege(privilege string) bool {
	return mysqlfamily.HasPrivilege(m.Privileges, privilege)
}
