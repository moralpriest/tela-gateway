package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/civilware/Gnomon/indexer"
	"github.com/civilware/Gnomon/storage"
	"github.com/civilware/Gnomon/structures"
	"github.com/civilware/tela"
	"github.com/civilware/tela/logger"
	"github.com/civilware/tela/shards"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
)

const gnomonSearchFilterINDEX = `Function init() Uint64
10 IF EXISTS("owner") == 0 THEN GOTO 30
20 RETURN 1
30 STORE("owner", address())
50 STORE("telaVersion", "1`

func main() {
	endpoint := daemonEndpoint()
	outPath := outputPath()

	// Gnomon stores its index under a shard dir next to the binary.
	dir, _ := os.Getwd()
	gnomonPath := filepath.Join(dir, "datashards", "gnomon")
	boltDB, err := storage.NewBBoltDB(gnomonPath, "gnomon")
	if err != nil {
		log.Fatalf("Gnomon boltdb: %v", err)
	}
	gravDB, _ := storage.NewGravDB(gnomonPath, "25ms")
	dbType := shards.GetDBType()

	var height int64
	switch dbType {
	case "boltdb":
		height, _ = boltDB.GetLastIndexHeight()
	default:
		height, _ = gravDB.GetLastIndexHeight()
	}

	exclusions := []string{"bb43c3eb626ee767c9f305772a6666f7c7300441a0ad8538a0799eb4f12ebcd2"}
	filter := []string{gnomonSearchFilterINDEX}

	idx := indexer.NewIndexer(gravDB, boltDB, dbType, filter, height, endpoint, "daemon", false, false, &structures.FastSyncConfig{
		Enabled:           false,
		SkipFSRecheck:     false,
		ForceFastSync:     false,
		ForceFastSyncDiff: 100,
		NoCode:            false,
	}, exclusions)

	indexer.InitLog(globals.Arguments, os.Stdout)

	if err := walletapi.Connect(endpoint); err != nil {
		log.Fatalf("connect daemon %s: %v", endpoint, err)
	}

	go idx.StartDaemonMode(10)

	// Wait for the scan to reach chain height (bounded).
	deadline := time.Now().Add(60 * time.Minute)
	for {
		daemonH := walletapi.Get_Daemon_Height()
		if idx.LastIndexedHeight >= daemonH && daemonH > 0 {
			break
		}
		if time.Now().After(deadline) {
			log.Printf("scan timeout; indexed %d/%d — writing partial list", idx.LastIndexedHeight, daemonH)
			break
		}
		time.Sleep(5 * time.Second)
	}

	all := idx.BBSBackend.GetAllOwnersAndSCIDs()
	aliases := map[string]string{}
	count := 0
	for scid := range all {
		info, err := tela.GetINDEXInfo(scid, endpoint)
		if err != nil || info.DURL == "" {
			continue
		}
		durl := strings.ToLower(strings.TrimSpace(info.DURL))
		if durl == "" {
			continue
		}
		// Short alias (strip .tela suffix) + full dURL both map to the SCID.
		aliases[durl] = scid
		if short := strings.TrimSuffix(durl, ".tela"); short != durl {
			aliases[short] = scid
		}
		count++
	}
	idx.Close()

	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", outPath, err)
	}
	fmt.Printf("wrote %d TELA INDEX apps to %s\n", count, outPath)
}

func daemonEndpoint() string {
	if v := os.Getenv("DERO_DAEMON_URLS"); v != "" {
		return strings.Split(v, ",")[0]
	}
	if v := os.Getenv("DERO_DAEMON_URL"); v != "" {
		return v
	}
	return "node.derofoundation.org:11012"
}

func outputPath() string {
	if v := os.Getenv("ALIASES_OUT"); v != "" {
		return v
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "aliases.json")
}

var _ = logger.Printf
