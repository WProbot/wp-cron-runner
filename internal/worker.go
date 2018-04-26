package internal

import (
	"context"
	"fmt"
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
		log.Println(fmt.Sprintf("Terminated worker [%d]", w.id))

		wg.Done()
	}()

	wg.Add(1)

	log.Println(fmt.Sprintf("Started worker [%d]", w.id))

	for {
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
		log.Println(fmt.Sprintf("[FAILED] (worker: %d) Adding CAPI event to site %s, error: %s", w.id, site, err))
	} else {
		log.Println(fmt.Sprintf("[  OK  ] (worker: %d) Adding CAPI event to site %s", w.id, site))
	}

	if err := w.cli.RunCron(site); err != nil {
		log.Println(fmt.Sprintf("[FAILED] (worker: %d) Running cron on site %s, error: %s", w.id, site, err))
	} else {
		log.Println(fmt.Sprintf("[  OK  ] (worker: %d) Running cron on site %s", w.id, site))
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
