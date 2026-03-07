package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"goyais/services/hub/internal/domain"

	_ "modernc.org/sqlite"
)

type memorySessionStore struct {
	mu       sync.Mutex
	sessions map[domain.SessionID]domain.Session
}

func newMemorySessionStore() *memorySessionStore {
	return &memorySessionStore{
		sessions: map[domain.SessionID]domain.Session{},
	}
}

func (s *memorySessionStore) Save(session domain.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

func BenchmarkSessionRepositorySave(b *testing.B) {
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)
	sessionFor := func(i int) domain.Session {
		return domain.Session{
			ID:                    domain.SessionID(fmt.Sprintf("sess_%d", i)),
			WorkspaceID:           domain.WorkspaceID("ws_local"),
			ProjectID:             "project_alpha",
			Name:                  fmt.Sprintf("Session %d", i),
			DefaultMode:           "default",
			ModelConfigID:         "model_default",
			WorkingDir:            "/tmp/project",
			AdditionalDirectories: []string{"/tmp/project/docs", "/tmp/project/specs"},
			NextSequence:          int64(i),
			CreatedAt:             now,
			UpdatedAt:             now,
		}
	}

	b.Run("memory_map", func(b *testing.B) {
		store := newMemorySessionStore()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			store.Save(sessionFor(i))
		}
	})

	b.Run("sqlite_repo", func(b *testing.B) {
		db := openBenchmarkDB(b)
		repo := NewSessionRepository(db)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := repo.Save(ctx, sessionFor(i)); err != nil {
				b.Fatalf("save session %d: %v", i, err)
			}
		}
	})
}

func openBenchmarkDB(b *testing.B) *sql.DB {
	b.Helper()

	dbPath := filepath.Join(b.TempDir(), "benchmark.sqlite3")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		b.Fatalf("open benchmark sqlite: %v", err)
	}
	b.Cleanup(func() {
		_ = db.Close()
	})

	if err := NewMigrator().Apply(db); err != nil {
		b.Fatalf("apply benchmark migrations: %v", err)
	}
	return db
}
