package internal

import (
	"context"
	"log"
	"sync"
)

type worker struct {
	id    int
	cli   *WpCli
	queue <-chan string
}

func (w *worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		log.Printf("Terminated worker [%d]\n", w.id)

		wg.Done()
	}()

	wg.Add(1)

	log.Printf("Started worker [%d]\n", w.id)

	for {

		// Prioritise context done channel to avoid select statement choosing randomly
		// between the queue and the context channel as below.
		select {
		case <-ctx.Done():
			return

		default:
			// don't block, fall through, live happy life!
		}

		select {
		case site := <-w.queue:
			w.runCron(site)

		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) runCron(site string) {
	if err := w.cli.ScheduleCronEvent("sqs_capi_sync_background_update", site); err != nil {
		log.Printf("[FAILED] (worker: %d) Adding CAPI event to site %s, error: %s\n", w.id, site, err)
	} else {
		log.Printf("[  OK  ] (worker: %d) Adding CAPI event to site %s\n", w.id, site)
	}

	if err := w.cli.RunCron(site); err != nil {
		log.Printf("[FAILED] (worker: %d) Running cron on site %s, error: %s\n", w.id, site, err)
	} else {
		log.Printf("[  OK  ] (worker: %d) Running cron on site %s\n", w.id, site)
	}
}

// SpawnWorker creates and runs a worker in a goroutine
func SpawnWorker(id int, ctx context.Context, wg *sync.WaitGroup, cli *WpCli, queue <-chan string) {
	w := &worker{
		id,
		cli,
		queue,
	}

	go w.Run(ctx, wg)
}
