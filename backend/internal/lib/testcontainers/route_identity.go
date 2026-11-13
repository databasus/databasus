package containers

// RouteIdentity is what a cluster's stable address serves: the PostgreSQL node, its timeline and the
// cluster's system identifier.
type RouteIdentity struct {
	Node             string
	Timeline         int
	SystemIdentifier string
}
