package mysqlfamily

type GrantScope string

const (
	GrantScopeGlobal GrantScope = "global"
	GrantScopeSchema GrantScope = "schema"
	GrantScopeTable  GrantScope = "table"
)
