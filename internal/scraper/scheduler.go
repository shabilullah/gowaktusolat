package scraper

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Scheduler struct {
	pool     *sqlitex.Pool
	cron     *cron.Cron
	schedule string
	stop     chan struct{}
}

func NewScheduler(pool *sqlitex.Pool, schedule string) *Scheduler {
	return &Scheduler{
		pool:     pool,
		cron:     cron.New(),
		schedule: schedule,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	if s.schedule == "" {
		log.Printf("[scheduler] No SEEDER_SCHED configured — scheduler disabled")
		return
	}

	_, err := s.cron.AddFunc(s.schedule, func() {
		now := time.Now()
		log.Printf("[scheduler] Triggered at %s, scraping year %d", now.Format(time.RFC3339), now.Year())
		RunFullScrape(context.Background(), s.pool, now.Year())
	})
	if err != nil {
		log.Printf("[scheduler] Failed to add cron job: %v", err)
		return
	}

	s.cron.Start()
	log.Printf("[scheduler] Started with schedule: %s", s.schedule)
}

func (s *Scheduler) Stop() {
	close(s.stop)
	ctx := s.cron.Stop()
	<-ctx.Done()
}
