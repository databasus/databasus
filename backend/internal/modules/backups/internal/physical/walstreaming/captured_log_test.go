package walstreaming

import (
	"bytes"
	"sync"
)

type capturedLog struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (c *capturedLog) Write(p []byte) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.Write(p)
}

func (c *capturedLog) String() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.String()
}
