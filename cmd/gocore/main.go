// ═══════════════════════════════════════════════════════════════════════════
// Package main implements the consolidated GO-CORE-LAB command line interface.
// ═══════════════════════════════════════════════════
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/binary"
	"github.com/Ri4ards2006/go-core-lab/internal/binary/flx"
	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/internal/pipeline"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
	"github.com/Ri4ards2006/go-core-lab/pkg/report"
)

const Version = "v1.0.0-core"

func printUsage() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               ⚡ GO-CORE-LAB CONSOLIDATED TOOLCHAIN ⚡               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Version: %s (%s/%s)\n\n", Version, runtime.GOOS, runtime.GOARCH)
	fmt.Println("Usage: gocore <subsystem> <command> [options] [arguments]")
	fmt.Println()
	fmt.Println("Binary Forensics (bin):")
	fmt.Println("  bin elf <file>                           Inspect ELF headers & sections (zero-copy mmap)")
	fmt.Println("  bin flx <file> [header|pool|disasm]      Inspect flux-lang bytecode container & disasm")
	fmt.Println("  bin hex <file> [--offset N] [--len N]    Canonical 16-byte ANSI-colorized hex visualizer")
	fmt.Println()
	fmt.Println("Network & Hardware Telemetry (net):")
	fmt.Println("  net live [options]                       Launch concurrent worker-pool pipeline & dashboard")
	fmt.Println("  net capture <iface> [-w out.pcap]        Direct raw socket packet sniffer")
	fmt.Println("  net pcap <file.pcap>                     Parse & replay packets from PCAP capture")
	fmt.Println("  net serial <device> [baud] [mode]        Direct UART/Serial bus telemetry monitor")
	fmt.Println()
	fmt.Println("General:")
	fmt.Println("  version                                  Print version and architecture")
	fmt.Println("  help                                     Show this help message")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		printUsage()

	case "version", "-v", "--version":
		fmt.Printf("gocore %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)

	case "bin":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing binary command (elf, flx, hex)")
			printUsage()
			os.Exit(1)
		}
		runBinCommand(os.Args[2:])

	case "net":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing network command (live, capture, pcap, serial)")
			printUsage()
			os.Exit(1)
		}
		runNetCommand(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown subsystem %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runBinCommand(args []string) {
	cmd := args[0]

	switch cmd {
	case "elf":
		if len(args) < 2 {
			fmt.Println("Usage: gocore bin elf <file>")
			os.Exit(1)
		}
		path := args[1]
		elfFile, mf, err := binary.ParseELFMmap(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing ELF %q: %v\n", path, err)
			os.Exit(1)
		}
		if mf != nil {
			defer mf.Close()
		}
		elfFile.Print()
		elfFile.PrintSections()

	case "flx":
		if len(args) < 2 {
			fmt.Println("Usage: gocore bin flx <file> [header|pool|disasm]")
			os.Exit(1)
		}
		path := args[1]
		flxFile, mf, err := flx.ParseFLXMmap(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing .flx %q: %v\n", path, err)
			os.Exit(1)
		}
		if mf != nil {
			defer mf.Close()
		}

		if len(args) >= 3 {
			switch args[2] {
			case "header":
				flxFile.PrintHeader()
			case "pool":
				flxFile.Pool.Print()
			case "disasm":
				flx.PrintDisassembly(flxFile.Instructions)
			default:
				flxFile.PrintSummary()
			}
		} else {
			flxFile.PrintSummary()
		}

	case "hex":
		fs := flag.NewFlagSet("hex", flag.ExitOnError)
		offsetFlag := fs.Int64("offset", 0, "Starting byte offset")
		lenFlag := fs.Int("len", 0, "Length in bytes to dump (0 for all)")
		noColorFlag := fs.Bool("no-color", false, "Disable ANSI color coding")
		upperFlag := fs.Bool("upper", false, "Display uppercase hex")

		if err := fs.Parse(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		if fs.NArg() < 1 {
			fmt.Println("Usage: gocore bin hex [flags] <file>")
			os.Exit(1)
		}

		path := fs.Arg(0)
		opts := report.HexDumpOptions{
			Offset:   *offsetFlag,
			Length:   *lenFlag,
			Colorize: !*noColorFlag,
			UpperHex: *upperFlag,
		}

		output, err := report.HexDumpFile(path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %q: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Print(output)

	default:
		fmt.Fprintf(os.Stderr, "Unknown binary command %q\n", cmd)
		os.Exit(1)
	}
}

func runNetCommand(args []string) {
	cmd := args[0]

	switch cmd {
	case "pcap":
		if len(args) < 2 {
			fmt.Println("Usage: gocore net pcap <file.pcap>")
			os.Exit(1)
		}
		runParsePcap(args[1])

	case "capture":
		if len(args) < 2 {
			fmt.Println("Usage: gocore net capture <interface> [-w output.pcap]")
			os.Exit(1)
		}
		iface := args[1]
		pcapOut := ""
		if len(args) >= 4 && args[2] == "-w" {
			pcapOut = args[3]
		}
		runCapture(iface, pcapOut)

	case "serial":
		if len(args) < 2 {
			fmt.Println("Usage: gocore net serial <device> [baud] [newline|sync]")
			os.Exit(1)
		}
		dev := args[1]
		baud := 115200
		mode := hw.DelimiterNewline
		if len(args) >= 3 {
			if b, err := strconv.Atoi(args[2]); err == nil {
				baud = b
			}
		}
		if len(args) >= 4 && args[3] == "sync" {
			mode = hw.DelimiterSyncByte
		}
		runSerial(dev, baud, mode)

	case "live":
		runLivePipeline(args[1:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown net command %q\n", cmd)
		os.Exit(1)
	}
}

func runLivePipeline(args []string) {
	fs := flag.NewFlagSet("live", flag.ExitOnError)
	ifaceFlag := fs.String("interface", "", "Network interface to sniff (e.g., eth0)")
	replayFlag := fs.String("replay", "", "Replay packets from PCAP file")
	serialFlag := fs.String("serial", "", "Serial device (e.g., /dev/ttyUSB0)")
	baudFlag := fs.Int("baud", 115200, "Serial baud rate")
	modeFlag := fs.String("mode", "newline", "Serial delimiter mode: newline or sync")
	workersFlag := fs.Int("workers", runtime.NumCPU(), "Dissection worker count")
	queueFlag := fs.Int("queue", 2048, "Queue buffer depth")
	pcapOutFlag := fs.String("pcap-out", "", "Output PCAP recording file")
	dashFlag := fs.Bool("dashboard", false, "Enable live terminal dashboard")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var src pipeline.Source
	if *ifaceFlag != "" {
		src = &rawSocketAdapter{iface: *ifaceFlag}
	} else if *replayFlag != "" {
		src = pipeline.NewPcapReplaySource(*replayFlag, true)
	} else if *serialFlag != "" {
		mode := hw.DelimiterNewline
		if *modeFlag == "sync" {
			mode = hw.DelimiterSyncByte
		}
		src = pipeline.NewSerialSource(hw.FrameConfig{
			Device:   *serialFlag,
			BaudRate: *baudFlag,
			Mode:     mode,
			SOF:      0xAA,
			EOF:      0x55,
		})
	} else {
		fmt.Println("Error: must specify --interface, --replay, or --serial")
		os.Exit(1)
	}

	cfg := pipeline.EngineConfig{
		NumWorkers:         *workersFlag,
		QueueSize:          *queueFlag,
		RingBufferSize:     512,
		DropOnBackpressure: false,
	}

	engine := pipeline.NewPipelineEngine(cfg, src)

	if *pcapOutFlag != "" {
		pSink, err := pipeline.NewPcapSink(*pcapOutFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating PCAP sink: %v\n", err)
			os.Exit(1)
		}
		engine.AddSink(pSink)
	}

	if !*dashFlag {
		engine.AddSink(&consoleLoggerSink{})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := engine.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting pipeline: %v\n", err)
		os.Exit(1)
	}

	if *dashFlag {
		dash := pipeline.NewDashboard(engine, 150*time.Millisecond)
		dash.Run(ctx)
	} else {
		<-ctx.Done()
	}

	fmt.Println("\n[*] Shutting down engine...")
	_ = engine.Stop()

	snap := engine.Stats.Snapshot()
	fmt.Printf("\n[Summary] Processed: %d packets (%.2f KiB) | Latency: %s\n",
		snap.TotalProcessed, float64(snap.TotalBytes)/1024, snap.AvgLatency)
}

type rawSocketAdapter struct {
	iface   string
	capture *gonet.RawSocketCapture
}

func (r *rawSocketAdapter) Start(ctx context.Context) (<-chan pipeline.IngestionEvent, error) {
	r.capture = gonet.NewRawSocketCapture(gonet.CaptureConfig{
		Interface: r.iface,
		SnapLen:   65535,
	})
	pktChan, err := r.capture.Start(ctx)
	if err != nil {
		return nil, err
	}

	outChan := make(chan pipeline.IngestionEvent, 128)
	go func() {
		defer close(outChan)
		var id uint64
		for pkt := range pktChan {
			id++
			outChan <- pipeline.IngestionEvent{
				ID:        id,
				Type:      pipeline.EventNetwork,
				Timestamp: pkt.Timestamp,
				Data:      pkt.Raw,
				Source:    r.iface,
			}
		}
	}()
	return outChan, nil
}

func (r *rawSocketAdapter) Close() error {
	if r.capture != nil {
		return r.capture.Close()
	}
	return nil
}

type consoleLoggerSink struct{}

func (c *consoleLoggerSink) OnEvent(ctx context.Context, event pipeline.PipelineEvent) error {
	fmt.Printf("#%04d [w:%d] %s\n", event.ID, event.WorkerID, event.Summary())
	return nil
}

func (c *consoleLoggerSink) Close() error {
	return nil
}

func runCapture(iface, pcapPath string) {
	fmt.Printf("[*] Live sniffing on %s...\n", iface)
	capEngine := gonet.NewRawSocketCapture(gonet.CaptureConfig{
		Interface: iface,
		SnapLen:   65535,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	packetChan, err := capEngine.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for pkt := range packetChan {
		count++
		fmt.Printf("#%04d %s\n", count, pkt.Summary())
	}
}

func runParsePcap(path string) {
	pr, f, err := export.OpenPcap(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	count := 0
	for {
		raw, ts, err := pr.NextPacket()
		if err != nil {
			break
		}
		count++
		pkt, _ := gonet.DissectWithTimestamp(raw, ts)
		if pkt != nil {
			fmt.Printf("#%04d %s\n", count, pkt.Summary())
		}
	}
	fmt.Printf("\n[*] Finished parsing %s. Total: %d packets.\n", path, count)
}

func runSerial(device string, baud int, mode hw.DelimiterMode) {
	fmt.Printf("[*] Serial monitor on %s (%d baud)...\n", device, baud)
	cfg := hw.FrameConfig{
		Device:   device,
		BaudRate: baud,
		Mode:     mode,
		SOF:      0xAA,
		EOF:      0x55,
	}
	monitor := hw.NewSerialMonitor(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	frameChan, err := monitor.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for frame := range frameChan {
		fmt.Println(frame.String())
	}
}
