package containers

import "context"

type NodeCommand struct {
	Node    string
	Command []string
}

func (cluster *CNPGCluster) ExecPostgresNodeCommand(ctx context.Context, command NodeCommand) (string, error) {
	output, err := cluster.K3s.Kubectl(ctx, KubectlCommand{Arguments: append(
		[]string{"-n", cluster.Namespace, "exec", command.Node, "-c", "postgres", "--"},
		command.Command...,
	)})

	return string(output), err
}
