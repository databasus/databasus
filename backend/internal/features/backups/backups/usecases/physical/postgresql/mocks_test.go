package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/walmath"
)

const testWalSegmentSize = int64(16 * 1024 * 1024)

// Used when a chain-span query must match every segment regardless of position.
const lsnSpanUpperBoundForTests = walmath.LSN(1) << 62

type mockWalStorage struct {
	mu        sync.Mutex
	saved     map[string][]byte
	deleted   []string
	saveCount atomic.Int64

	failSaveTimes int

	isFailingAllSaves atomic.Bool

	// Interleaves the DeleteFull cascade race: a save whose name contains this
	// substring signals started and waits on release before returning. Object names
	// carry a per-attempt UUID, so the test cannot know the whole name in advance.
	blockOn   string
	blockOnce sync.Once
	started   chan struct{}
	release   chan struct{}

	// Holds every segment artifact write, sidecars aside, until the test lets it through.
	artifactGate *artifactWriteGate
}

// Records which segment artifacts started writing, in order, and how many wrote at once.
type artifactWriteGate struct {
	mutex                   sync.Mutex
	startedObjectNames      []string
	activeWriteCount        int
	highestActiveWriteCount int

	releases         chan struct{}
	releaseAllOnce   sync.Once
	openAtWriteCount int
}

func newArtifactWriteGate() *artifactWriteGate {
	return &artifactWriteGate{releases: make(chan struct{})}
}

// The gate stays closed until this many artifacts write at once, then lets every write through, so a
// test can prove writes overlapped without releasing them one by one.
func newArtifactWriteGateOpeningAt(activeWriteCount int) *artifactWriteGate {
	gate := newArtifactWriteGate()
	gate.openAtWriteCount = activeWriteCount

	return gate
}

func (g *artifactWriteGate) passThrough(objectName string) {
	g.mutex.Lock()
	g.startedObjectNames = append(g.startedObjectNames, objectName)
	g.activeWriteCount++
	g.highestActiveWriteCount = max(g.highestActiveWriteCount, g.activeWriteCount)
	isOpeningCountReached := g.openAtWriteCount > 0 && g.activeWriteCount >= g.openAtWriteCount
	g.mutex.Unlock()

	if isOpeningCountReached {
		g.releaseAll()
	}

	<-g.releases

	g.mutex.Lock()
	g.activeWriteCount--
	g.mutex.Unlock()
}

func (g *artifactWriteGate) releaseOne() {
	g.releases <- struct{}{}
}

func (g *artifactWriteGate) releaseAll() {
	g.releaseAllOnce.Do(func() { close(g.releases) })
}

func (g *artifactWriteGate) getActiveWriteCount() int {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.activeWriteCount
}

func (g *artifactWriteGate) getHighestActiveWriteCount() int {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.highestActiveWriteCount
}

func (g *artifactWriteGate) getStartedObjectNames() []string {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return slices.Clone(g.startedObjectNames)
}

func (g *artifactWriteGate) countStartedWritesFor(walFilename string) int {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	count := 0

	for _, objectName := range g.startedObjectNames {
		if strings.Contains(objectName, walFilename) {
			count++
		}
	}

	return count
}

// hasObjectFor reports whether any stored object belongs to the given segment.
// A test cannot rebuild the exact key, which carries a per-attempt UUID, so it
// matches on the segment identity the key derives from.
func (m *mockWalStorage) hasObjectFor(databaseID uuid.UUID, timelineID int, walFilename string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefix := fmt.Sprintf("%s-WAL-tl%d-%s-", databaseID, timelineID, walFilename)

	for name := range m.saved {
		if strings.HasPrefix(name, prefix) && !strings.HasSuffix(name, metadataSuffix) {
			return true
		}
	}

	return false
}

func newMockWalStoreFor(store *mockWalStorage) *FakeFileStore {
	return NewFakeFileStore(store)
}

// objectNamesFor returns every object the mock holds for one segment, artifact and
// sidecar alike, so a test can name what it has to wait for.
func (m *mockWalStorage) objectNamesFor(databaseID uuid.UUID, timelineID int, walFilename string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefix := fmt.Sprintf("%s-WAL-tl%d-%s-", databaseID, timelineID, walFilename)

	names := make([]string, 0, 2)

	for name := range m.saved {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}

	return names
}

func newMockWalStorage() *mockWalStorage {
	return &mockWalStorage{saved: make(map[string][]byte)}
}

// newBlockingMockWalStorage holds the artifact write of the segment until the test closes release.
func newBlockingMockWalStorage(segmentName string) *mockWalStorage {
	store := newMockWalStorage()
	store.blockOn = segmentName
	store.started = make(chan struct{})
	store.release = make(chan struct{})

	return store
}

func (m *mockWalStorage) SaveFile(_ context.Context, fileName string, file io.Reader) error {
	m.saveCount.Add(1)

	body, _ := io.ReadAll(file)

	if m.artifactGate != nil && !strings.HasSuffix(fileName, metadataSuffix) {
		m.artifactGate.passThrough(fileName)
	}

	// The sidecar shares the artifact's name, so the block is scoped to the
	// artifact and fires once.
	if m.blockOn != "" && strings.Contains(fileName, m.blockOn) && !strings.HasSuffix(fileName, metadataSuffix) {
		m.blockOnce.Do(func() {
			close(m.started)
			<-m.release
		})
	}

	if m.isFailingAllSaves.Load() {
		return fmt.Errorf("mock storage is failing every save for %s", fileName)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failSaveTimes > 0 {
		m.failSaveTimes--

		return fmt.Errorf("mock storage induced failure for %s", fileName)
	}

	m.saved[fileName] = body

	return nil
}

func (m *mockWalStorage) startFailingSaves() {
	m.isFailingAllSaves.Store(true)
}

func (m *mockWalStorage) stopFailingSaves() {
	m.isFailingAllSaves.Store(false)
}

func (m *mockWalStorage) DeleteFile(_ context.Context, fileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.saved, fileName)
	m.deleted = append(m.deleted, fileName)

	return nil
}

func (m *mockWalStorage) hasObject(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.saved[name]

	return ok
}

// 256 segments per logid, i.e. the default 16 MB segment size.
func walName(timeline uint32, segNo uint64) string {
	const segmentsPerLogID = 256

	return fmt.Sprintf("%08X%08X%08X", timeline, segNo/segmentsPerLogID, segNo%segmentsPerLogID)
}

// The segments are sparse: only the reported size matters to the uploader and to
// the resume-point math, so allocating real bytes would only slow tests down.
func writeSegmentOfSize(t *testing.T, dir, name string, sizeBytes int64) string {
	t.Helper()

	path := filepath.Join(dir, name)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	require.NoError(t, err)

	require.NoError(t, file.Truncate(sizeBytes))
	require.NoError(t, file.Close())

	return path
}

func writeWalFile(t *testing.T, dir, name string) string {
	t.Helper()

	return writeSegmentOfSize(t, dir, name, testWalSegmentSize)
}
