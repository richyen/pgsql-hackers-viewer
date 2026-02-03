package api

import (
	"sync"
	"time"

	"github.com/pgsql-analyzer/backend/models"
)

// GlobalSyncState tracks the current sync progress
var GlobalSyncState = &SyncState{
	Progress: models.SyncProgress{
		MonthsSynced: 0,
		TotalMonths:  0,
		IsSyncing:    false,
	},
}

type SyncState struct {
	mu       sync.RWMutex
	Progress models.SyncProgress
}

func (s *SyncState) Update(monthsSynced, totalMonths int, currentMonth string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Progress.MonthsSynced = monthsSynced
	s.Progress.TotalMonths = totalMonths
	s.Progress.CurrentMonth = currentMonth
	now := time.Now()
	s.Progress.LastSyncedAt = &now
}

func (s *SyncState) SetSyncing(syncing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Progress.IsSyncing = syncing
}

func (s *SyncState) SetLatestMessageDate(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Progress.LatestMessageDate = &t
}

func (s *SyncState) Get() models.SyncProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Progress
}
