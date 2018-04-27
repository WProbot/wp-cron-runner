package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"

	"cron-runner/internal"
)

var (
	wpCliPath  = "/usr/local/bin/wp"
	wpPath     = "/srv/www/wp"
	maxWorkers = 5
	queueSize  = 100
)

func parseArgs() error {
	var err error

	if value, ok := os.LookupEnv("CRON_RUNNER_WP_PATH"); ok {
		wpPath = value
	}

	if value, ok := os.LookupEnv("CRON_RUNNER_WP_CLI_PATH"); ok {
		wpCliPath = value
	}

	if value, ok := os.LookupEnv("CRON_RUNNER_QUEUE_SIZE"); ok {
		q, err := strconv.Atoi(value)

		if err != nil {
			return fmt.Errorf("invalid value provided for CRON_RUNNER_QUEUE_SIZE environment variable: %s", err)
		}

		queueSize = q
	}

	if value, ok := os.LookupEnv("CRON_RUNNER_MAX_WORKERS"); ok {
		q, err := strconv.Atoi(value)

		if err != nil {
			return fmt.Errorf("invalid value provided for CRON_RUNNER_MAX_WORKERS environment variable: %s", err)
		}

		maxWorkers = q
	}

	flag.StringVar(&wpPath, "path", wpPath, "WordPress installation directory")
	flag.StringVar(&wpCliPath, "wp-cli", wpCliPath, "Path to WP CLI binary")
	flag.IntVar(&queueSize, "queue", queueSize, "Maximum job queue size")
	flag.IntVar(&maxWorkers, "workers", maxWorkers, "Maximum number or workers to spawn")
	flag.Parse()

	if wpCliPath, err = validatePath(wpCliPath); err != nil {
		return fmt.Errorf("invalid argument \"wp-cli\", %s", err)
	}

	if wpPath, err = validatePath(wpPath); err != nil {
		return fmt.Errorf("invalid argument \"path\", %s", err)
	}

	if queueSize <= 0 {
		return errors.New("invalid argument \"queue\": must be greater than zero")
	}

	if maxWorkers <= 0 {
		return errors.New("invalid argument \"workers\": must be greater than zero")
	} else if runtime.NumCPU() == 1 && maxWorkers > 1 {

		// Here we have a problem: WP CLI is a real CPU hog!
		//
		// Running this app on a single core with multiple workers does not have any benefits.
		// More over it maxes out CPU and because we are running in a auto scaling group,
		// it starts adding new servers.
		//
		// So, the solution is to limit the number of workers to 1 on a single CPU core.

		maxWorkers = 1

		log.Println("Single CPU detected and the number of workers is limited to 1")
	}

	return nil
}

func validatePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("required and must not be empty")
	}

	path, err := filepath.Abs(path)

	if err != nil {
		return "", err
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("%s: no such file or directory", path)
	}

	return path, nil
}

func main() {
	log.SetOutput(os.Stdout)

	// Check arguments

	if err := parseArgs(); err != nil {
		log.Fatalln("Error:", err)
	}

	// Check WP CLI and WP Core versions to make sure that WP CLI works
	// and WP Core installation is valid

	cli := internal.NewWpCli(wpCliPath, wpPath)
	version, err := cli.Version()

	if err != nil {
		log.Fatalln("Error:", err)
	}

	log.Println("WP CLI version:", version)

	version, err = cli.CoreVersion()

	if err != nil {
		log.Fatalln("Error:", err)
	}

	log.Println("WP Core version:", version)

	// Setup signal listener

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)

	// Start runner

	log.Println("Starting cron runner ...")

	wg := new(sync.WaitGroup)
	ctx, cancelFunc := context.WithCancel(context.Background())
	queue := make(chan string, queueSize)

	internal.SpawnScheduler(ctx, wg, &cli, queue)

	for i := 0; i < maxWorkers; i++ {
		internal.SpawnWorker(i+1, ctx, wg, &cli, queue)
	}

	<-exit // block until exit signal

	log.Println("Gracefully shutting down ...")

	cancelFunc()

	wg.Wait()

	close(queue)

	log.Println("Bye, bye!")
}
