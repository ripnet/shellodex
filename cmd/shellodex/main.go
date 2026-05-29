package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ripnet/shellodex/internal/config"
	"github.com/ripnet/shellodex/internal/connect"
	"github.com/ripnet/shellodex/internal/model"
	"github.com/ripnet/shellodex/internal/tui"
	"github.com/ripnet/shellodex/internal/version"
)

func main() {
	var cfgPath string
	var showVersion bool
	var doUpdate bool

	flag.StringVar(&cfgPath, "config", "", "path to config file (default: platform config dir)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&doUpdate, "update", false, "download and install the latest release")
	flag.Parse()

	if showVersion {
		fmt.Printf("shellodex %s (%s) built %s\n", version.Version, version.Commit, version.BuildDate)
		os.Exit(0)
	}

	if doUpdate {
		fmt.Printf("Checking for updates (current: %s)...\n", version.Version)
		newVer, err := version.SelfUpdate(version.Version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "shellodex: update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Updated to %s — restart shellodex to use the new version.\n", newVer)
		os.Exit(0)
	}

	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "shellodex: cannot determine config path: %v\n", err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shellodex: cannot load config %s: %v\n", cfgPath, err)
		os.Exit(1)
	}

	saveFn := func(c *model.Config) error {
		return config.Save(cfgPath, c)
	}

	app := tui.NewAppModel(cfg, cfgPath, saveFn)

	p := tea.NewProgram(app, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		log.Fatalf("shellodex: %v", err)
	}

	// After the TUI exits, check for a pending connect request.
	// connect.Connect exec-replaces this process with ssh/telnet on success.
	if final, ok := result.(tui.AppModel); ok {
		if req := final.ConnectRequest; req != nil {
			// Stamp the host's last-connected time before exec-replacing the process.
			now := time.Now()
			for i := range cfg.Hosts {
				if cfg.Hosts[i].ID == req.Host.ID {
					cfg.Hosts[i].LastConnected = &now
					_ = config.Save(cfgPath, cfg)
					break
				}
			}
			if connErr := connect.Connect(&req.Host, cfg); connErr != nil {
				fmt.Fprintf(os.Stderr, "shellodex: connect failed: %v\n", connErr)
				os.Exit(1)
			}
		}
	}
}
