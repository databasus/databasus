package sshtunnel

import (
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	keepaliveInterval = 30 * time.Second
	keepaliveTimeout  = 15 * time.Second
)

func (f *Forwarder) runKeepalive() {
	ticker := time.NewTicker(keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.done:
			return
		case <-ticker.C:
			f.clientMutex.Lock()
			client := f.client
			f.clientMutex.Unlock()

			if client == nil {
				continue
			}

			if !f.isKeepaliveAnsweredWithin(keepaliveTimeout, client) {
				f.logger.WarnContext(f.lifetime, "ssh tunnel keepalive went unanswered, reconnecting on next use")
				f.discardClient(client)
			}
		}
	}
}

// The budget is a parameter because the reachability probe answers a caller with a far shorter
// deadline than the keepalive loop's.
//
// ssh.SendRequest takes no timeout and blocks forever on a connection that died without a FIN -
// the exact failure keepalive exists to detect - so it must never run on a goroutine that Close
// waits for. The orphaned goroutine unblocks once discardClient closes the client.
func (f *Forwarder) isKeepaliveAnsweredWithin(budget time.Duration, client *ssh.Client) bool {
	keepaliveAnswer := make(chan error, 1)

	go func() {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		keepaliveAnswer <- err
	}()

	timer := time.NewTimer(budget)
	defer timer.Stop()

	select {
	case err := <-keepaliveAnswer:
		return err == nil
	case <-timer.C:
		return false
	case <-f.done:
		return false
	}
}
