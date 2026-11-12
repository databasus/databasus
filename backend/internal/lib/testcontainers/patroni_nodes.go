package containers

func (fixture *Patroni) GetNode(name string) PatroniNode {
	for _, node := range fixture.Nodes {
		if node.Name == name {
			return node
		}
	}

	panic("unknown Patroni node: " + name)
}

func (fixture *Patroni) GetReplica(primary string) PatroniNode {
	for _, node := range fixture.Nodes {
		if node.Name != primary {
			return node
		}
	}

	panic("Patroni replica unavailable")
}
