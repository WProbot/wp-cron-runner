package internal

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"
)

type scheduler struct {
	cli   *WpCli
	sites []string
	queue chan<- string
}

func (s *scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()

		log.Println("Terminated scheduler")
	}()

	wg.Add(1)

	log.Println("Started scheduler")

	if err := s.refreshSites(); err == nil {
		s.requeueSites()
	}

	requeueTimer := time.NewTicker(1 * time.Second) // small delay between adding jobs to the queue
	refreshTimer := time.NewTicker(time.Minute)

	for {
		select {
		case <-ctx.Done():
			requeueTimer.Stop()
			refreshTimer.Stop()
			return

		case <-requeueTimer.C:
			s.requeueSites()

		case <-refreshTimer.C:
			s.refreshSites()
		}
	}
}

func (s *scheduler) refreshSites() error {
	rand.Seed(time.Now().UTC().UnixNano())

	urls, err := s.cli.SiteUrls()

	if err != nil {
		log.Println("[FAILED] Refreshing sites list:", err)

		return err
	}

	// Shuffle URLs for randomness

	for i := range urls {
		j := rand.Intn(i + 1)
		urls[i], urls[j] = urls[j], urls[i]
	}

	s.sites = urls

	log.Println("[  OK  ] Refreshing sites list")

	return nil
}

func (s *scheduler) requeueSites() {
	if len(s.queue) == 0 {
		for _, site := range s.sites {
			s.queue <- site
		}

		log.Println("[  OK  ] Adding new jobs to the site queue")
	}
}

// SpawnScheduler creates and runs a scheduler in a goroutine
func SpawnScheduler(ctx context.Context, wg *sync.WaitGroup, cli *WpCli, queue chan<- string) {
	sites := make([]string, 0)

	s := &scheduler{
		cli,
		sites,
		queue,
	}

	go s.Run(ctx, wg)
}
