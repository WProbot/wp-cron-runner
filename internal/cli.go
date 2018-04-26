package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// wpCli structure contains pre-built arguments required to run WP CLI
// and provides API to interact with it.
type WpCli struct {
	wpCli string
	args  []string
}

func NewWpCli(wpCliPath, wpPath string) *WpCli {
	args := []string{
		fmt.Sprintf("--path=%s", wpPath),
		"--quiet",
		"--allow-root",
	}

	return &WpCli{
		wpCliPath,
		args,
	}
}

// Run executes WP CLI with provided subcommands and returns command output
func (cli WpCli) Run(subcommands []string) ([]byte, error) {
	var out []byte

	args := append(subcommands, cli.args...)

	cmd := exec.Command(cli.wpCli, args...)
	out, err := cmd.Output()

	return out, err
}

// Version returns WP CLI version number
func (cli *WpCli) Version() (string, error) {
	out, err := cli.Run([]string{"cli", "version"})

	return strings.TrimSpace(string(out)), err
}

// CoreVersion returns WordPress core version number
func (cli *WpCli) CoreVersion() (string, error) {
	out, err := cli.Run([]string{"core", "version"})

	return strings.TrimSpace(string(out)), err
}

// SiteUrls returns a list of URLs of the WordPress sites on multi-site installation
func (cli *WpCli) SiteUrls() ([]string, error) {
	out, err := cli.Run([]string{
		"site",
		"list",
		"--fields=domain,url",
		"--archived=false",
		"--deleted=false",
		"--spam=false",
		"--format=json",
	})

	if err != nil {
		return nil, err
	}

	raw := make([]struct {
		Domain string `json:"domain"`
		Url    string `json:"url"`
	}, 0)

	if err = json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}

	list := make([]string, 0)

	for _, entry := range raw {
		parsed, err := url.Parse(entry.Url)

		if err != nil || parsed.Hostname() == "vip.local" || entry.Domain == "vip.local" {
			continue // skip vip.local (default site)
		}

		list = append(list, entry.Url)
	}

	return list, nil
}

// ScheduleCronEvent schedules the given cron event for immediate run
func (cli *WpCli) ScheduleCronEvent(event, site string) error {
	_, err := cli.Run([]string{"cron", "event", "schedule", event, "now", fmt.Sprintf("--url=%s", site)})

	return err
}

// RunCron executes WordPress cron
func (cli *WpCli) RunCron(site string) error {
	_, err := cli.Run([]string{"cron", "event", "run", "--due-now", fmt.Sprintf("--url=%s", site)})

	return err
}
