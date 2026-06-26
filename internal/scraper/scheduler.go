package scraper

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	db   *sql.DB
	cron *cron.Cron
	stop chan struct{}
}

func NewScheduler(db *sql.DB) *Scheduler {
	return &Scheduler{
		db:   db,
		cron: cron.New(),
		stop: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stop)
	ctx := s.cron.Stop()
	<-ctx.Done()
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	var currentSchedule string
	var currentEnabled bool

	checkSettings := func() {
		schedule, enabled := s.readSettings()
		if schedule != currentSchedule || enabled != currentEnabled {
			s.reconfigure(schedule, enabled)
			currentSchedule = schedule
			currentEnabled = enabled
		}
	}

	checkSettings()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			checkSettings()
		}
	}
}

func (s *Scheduler) readSettings() (schedule string, enabled bool) {
	var val string
	if err := s.db.QueryRow("SELECT value FROM settings WHERE key = 'scraper.schedule'").Scan(&val); err != nil {
		log.Printf("[scheduler] Failed to read schedule: %v", err)
		return "", false
	}
	schedule = val

	if err := s.db.QueryRow("SELECT value FROM settings WHERE key = 'scraper.enabled'").Scan(&val); err != nil {
		log.Printf("[scheduler] Failed to read enabled: %v", err)
		return schedule, false
	}
	enabled = val == "true"

	return
}

func (s *Scheduler) reconfigure(schedule string, enabled bool) {
	// Remove all existing jobs
	for _, entry := range s.cron.Entries() {
		s.cron.Remove(entry.ID)
	}

	if !enabled {
		log.Printf("[scheduler] Scraper disabled")
		return
	}

	if schedule == "" {
		log.Printf("[scheduler] No schedule configured")
		return
	}

	_, err := s.cron.AddFunc(schedule, func() {
		now := time.Now()
		log.Printf("[scheduler] Triggered at %s, scraping year %d", now.Format(time.RFC3339), now.Year())
		RunFullScrape(context.Background(), s.db, now.Year())
	})
	if err != nil {
		log.Printf("[scheduler] Failed to add cron job: %v", err)
		return
	}

	s.cron.Start()
	log.Printf("[scheduler] Enabled with schedule: %s", schedule)
}
