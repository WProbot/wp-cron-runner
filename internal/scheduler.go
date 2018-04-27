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
	wg.Add(1)

	log.Println("Started scheduler")

	if err := s.refreshSites(); err == nil {
		s.requeueSites()
	}

	requeueTimer := time.NewTicker(5 * time.Second) // small delay between adding jobs to the queue
	refreshTimer := time.NewTicker(time.Minute)

	defer func() {
		log.Println("Terminated scheduler")

		requeueTimer.Stop()
		refreshTimer.Stop()

		wg.Done()
	}()

	for {
		select {
		case <-requeueTimer.C:
			s.requeueSites()

		case <-refreshTimer.C:
			s.refreshSites()

		case <-ctx.Done():
			return
		}
	}
}

func (s *scheduler) refreshSites() error {
	urls, err := s.cli.SiteUrls()

	if err != nil {
		log.Println("[FAILED] Refreshing sites list:", err)

		return err
	}

	rand.Seed(time.Now().UTC().UnixNano())

	// Shuffle URLs for randomness and to minimise possible collisions,
	// when there are multiple instances running
	rand.Shuffle(len(urls), func(i, j int) {
		urls[i], urls[j] = urls[j], urls[i]
	})

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
