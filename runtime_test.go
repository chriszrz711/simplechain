package main

import (
	"io"
	"reflect"
	"testing"
)

func TestRunRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    *Node
		addr string
	}{
		{"nil_node", nil, ":0"}, {"nil_chain", NewNode(nil, NewMempool(), "", nil), ":0"},
		{"empty_chain", NewNode(&Blockchain{}, NewMempool(), "", nil), ":0"},
		{"nil_tip", NewNode(&Blockchain{Blocks: []*Block{nil}}, NewMempool(), "", nil), ":0"},
		{"nil_mempool", NewNode(NewBlockchain(), nil, "", nil), ":0"},
		{"empty_address", NewNode(NewBlockchain(), NewMempool(), "", nil), ""},
		{"invalid_address", NewNode(NewBlockchain(), NewMempool(), "", nil), "127.0.0.1:invalid-port"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.n.Run(tc.addr); err == nil {
				t.Fatal("expected server configuration error")
			}
		})
	}
}

func TestParseNodeConfig(t *testing.T) {
	cfg, err := parseNodeConfig([]string{"-port", "8002", "-miner", "Alice", "-peers", " http://localhost:8001, ,http://localhost:8003/ "}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != 8002 || cfg.miner != "Alice" || !reflect.DeepEqual(cfg.peers, []string{"http://localhost:8001", "http://localhost:8003/"}) {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
	defaults, err := parseNodeConfig([]string{"-miner", "Miner"}, io.Discard)
	if err != nil || defaults.port != 8001 || len(defaults.peers) != 0 {
		t.Fatalf("unexpected defaults: %+v %v", defaults, err)
	}
	for _, args := range [][]string{{}, {"-miner", " "}, {"-miner", "Alice", "-port", "0"}, {"-miner", "Alice", "-port", "65536"}, {"-miner", "Alice", "-port", "abc"}, {"-unknown"}, {"-miner", "Alice", "extra"}} {
		if _, err := parseNodeConfig(args, io.Discard); err == nil {
			t.Fatalf("expected invalid flags: %v", args)
		}
	}
}
