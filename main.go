package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type nodeConfig struct {
	port  int
	miner string
	peers []string
}

func parseNodeConfig(args []string, output io.Writer) (nodeConfig, error) {
	var cfg nodeConfig
	var peers string
	flags := flag.NewFlagSet("simplechain", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.IntVar(&cfg.port, "port", 8001, "HTTP listening port (1-65535)")
	flags.StringVar(&cfg.miner, "miner", "", "miner reward address (required)")
	flags.StringVar(&peers, "peers", "", "comma-separated static peer HTTP URLs")
	if err := flags.Parse(args); err != nil {
		return nodeConfig{}, err
	}
	if flags.NArg() != 0 {
		return nodeConfig{}, fmt.Errorf("unexpected positional arguments")
	}
	if cfg.port < 1 || cfg.port > 65535 {
		return nodeConfig{}, fmt.Errorf("port must be between 1 and 65535")
	}
	cfg.miner = strings.TrimSpace(cfg.miner)
	if cfg.miner == "" {
		return nodeConfig{}, fmt.Errorf("-miner is required")
	}
	for _, peer := range strings.Split(peers, ",") {
		if peer = strings.TrimSpace(peer); peer != "" {
			cfg.peers = append(cfg.peers, peer)
		}
	}
	return cfg, nil
}

func main() {
	cfg, err := parseNodeConfig(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	node := NewNode(NewBlockchain(), NewMempool(), cfg.miner, cfg.peers)
	addr := fmt.Sprintf(":%d", cfg.port)
	fmt.Fprintf(os.Stderr, "Simplechain listening on %s (fresh in-memory chain)\n", addr)
	if err := node.Run(addr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
