package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/internal/pipeline"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

func printUsage() {
	fmt.Println("Usage: net-probe <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  live [options]                           Launch concurrent worker-pool pipeline engine")
	fmt.Println("  capture <interface> [-w output.pcap]     Direct raw socket packet sniffer")
	fmt.Println("  parse-pcap <file.pcap>                   Parse and replay packets from a PCAP file")
	fmt.Println("  serial <device> [baud] [newline|sync]    Direct UART/Serial bus telemetry monitor")
	fmt.Println("  help                                     Show this help message")
	fmt.Println()
	fmt.Println("Options for 'live':")
	fmt.Println("  --interface <iface>      Network interface to sniff (e.g., eth0, lo)")
	fmt.Println("  --replay <file.pcap>     Replay packets from PCAP file")
	fmt.Println("  --serial <device>        Serial device (e.g., /dev/ttyUSB0)")
	fmt.Println("  --baud <rate>            Serial baud rate (default: 115200)")
	fmt.Println("  --mode <newline|sync>    Serial delimiter mode (default: newline)")
	fmt.Println("  --workers <n>            Number of dissection workers (default: NumCPU)")
	fmt.Println("  --queue <size>           Ingestion queue buffer size (default: 2048)")
	fmt.Println("  --pcap-out <file.pcap>   Record processed packets to PCAP file")
	fmt.Println("  --dashboard              Enable real-time ANSI terminal telemetry dashboard")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "help", "-h", "--help":
		printUsage()

	case "live":
		runLivePipeline(os.Args[2:])

	case "capture":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing interface for capture")
			fmt.Println("Usage: net-probe capture <interface> [-w output.pcap]")
			os.Exit(1)
		}
		iface := os.Args[2]
		pcapOut := ""
		if len(os.Args) >= 5 && os.Args[3] == "-w" {
			pcapOut = os.Args[4]
		}
		runCapture(iface, pcapOut)

	case "parse-pcap":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing pcap file")
			fmt.Println("Usage: net-probe parse-pcap <file.pcap>")
			os.Exit(1)
		}
		runParsePcap(os.Args[2])

	case "serial":
		if len(os.Args) < 3 {
			fmt.Println("Error: missing serial device")
			fmt.Println("Usage: net-probe serial <device> [baud] [newline|sync]")
			os.Exit(1)
		}
		dev := os.Args[2]
		baud := 115200
		mode := hw.DelimiterNewline
		if len(os.Args) >= 4 {
			if b, err := strconv.Atoi(os.Args[3]); err == nil {
				baud = b
			}
		}
		if len(os.Args) >= 5 && os.Args[4] == "sync" {
			mode = hw.DelimiterSyncByte
		}
		runSerial(dev, baud, mode)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n\n", cmd)
		printUsage()
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
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// 1. Determine Source
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
		fmt.Println("Error: must specify one of --interface, --replay, or --serial")
		fmt.Println("Run 'net-probe live --help' for options.")
		os.Exit(1)
	}

	// 2. Configure Pipeline Engine
	cfg := pipeline.EngineConfig{
		NumWorkers:         *workersFlag,
		QueueSize:          *queueFlag,
		RingBufferSize:     512,
		DropOnBackpressure: false,
	}

	engine := pipeline.NewPipelineEngine(cfg, src)

	// 3. Optional PCAP Sink
	if *pcapOutFlag != "" {
		pSink, err := pipeline.NewPcapSink(*pcapOutFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating PCAP sink: %v\n", err)
			os.Exit(1)
		}
		engine.AddSink(pSink)
		fmt.Printf("[*] Recording pipeline events to PCAP: %s\n", *pcapOutFlag)
	}

	// 4. Console Logger Sink if dashboard disabled
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
		fmt.Fprintf(os.Stderr, "Error starting pipeline engine: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[*] Pipeline Engine started with %d workers (queue size: %d)\n", cfg.NumWorkers, cfg.QueueSize)

	if *dashFlag {
		dash := pipeline.NewDashboard(engine, 150*time.Millisecond)
		dash.Run(ctx)
	} else {
		<-ctx.Done()
	}

	// Graceful Teardown
	fmt.Println("\n[*] Shutting down pipeline engine...")
	_ = engine.Stop()

	// Print Final Summary Report
	snap := engine.Stats.Snapshot()
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   FINAL EXECUTION SUMMARY                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Total Processed:  %d events\n", snap.TotalProcessed)
	fmt.Printf("  Total Volume:     %.2f KiB\n", float64(snap.TotalBytes)/1024)
	fmt.Printf("  Dropped Events:   %d\n", snap.DroppedEvents)
	fmt.Printf("  Error Events:     %d\n", snap.ErrorEvents)
	fmt.Printf("  Avg Latency:      %s (Min: %s, Max: %s)\n", snap.AvgLatency, snap.MinLatency, snap.MaxLatency)
	fmt.Printf("  Protocol Stats:   IPv4: %d, IPv6: %d, ARP: %d, TCP: %d, UDP: %d, ICMP: %d, Serial: %d\n",
		snap.IPv4Count, snap.IPv6Count, snap.ARPCount, snap.TCPCount, snap.UDPCount, snap.ICMPCount, snap.SerialCount)
}

// rawSocketAdapter adapts RawSocketCapture to the pipeline.Source interface.
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

// consoleLoggerSink formats and logs events to standard output.
type consoleLoggerSink struct{}

func (c *consoleLoggerSink) OnEvent(ctx context.Context, event pipeline.PipelineEvent) error {
	fmt.Printf("#%04d [w:%d] %s\n", event.ID, event.WorkerID, event.Summary())
	if event.Packet != nil && len(event.Packet.Payload) > 0 {
		fmt.Print(gonet.FormatHexDump(event.Packet.Payload))
	}
	return nil
}

func (c *consoleLoggerSink) Close() error {
	return nil
}

func runCapture(iface string, pcapPath string) {
	fmt.Printf("[*] Starting live packet capture on %s...\n", iface)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[*] Stopping capture...")
		cancel()
	}()

	var pcapWriter *export.PcapWriter
	if pcapPath != "" {
		f, err := os.Create(pcapPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating PCAP file %q: %v\n", pcapPath, err)
			os.Exit(1)
		}
		defer f.Close()

		pw, err := export.NewPcapWriter(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing PCAP writer: %v\n", err)
			os.Exit(1)
		}
		pcapWriter = pw
		fmt.Printf("[*] Recording packets to %s\n", pcapPath)
	}

	capEngine := gonet.NewRawSocketCapture(gonet.CaptureConfig{
		Interface: iface,
		SnapLen:   65535,
	})

	packetChan, err := capEngine.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting capture: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for pkt := range packetChan {
		count++
		fmt.Printf("#%04d %s\n", count, pkt.Summary())
		if len(pkt.Payload) > 0 {
			fmt.Print(gonet.FormatHexDump(pkt.Payload))
		}

		if pcapWriter != nil {
			_ = pcapWriter.WritePacket(pkt.Raw, pkt.Timestamp)
		}
	}

	fmt.Printf("[*] Total packets captured: %d\n", count)
}

func runParsePcap(path string) {
	fmt.Printf("[*] Parsing PCAP file: %s\n\n", path)

	pr, f, err := export.OpenPcap(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening PCAP: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	count := 0
	for {
		raw, ts, err := pr.NextPacket()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "Error reading packet #%d: %v\n", count+1, err)
			break
		}

		count++
		pkt, err := gonet.DissectWithTimestamp(raw, ts)
		if err != nil {
			fmt.Printf("#%04d [%s] (Corrupted frame: %v)\n", count, ts.Format("15:04:05.000000"), err)
			continue
		}

		fmt.Printf("#%04d %s\n", count, pkt.Summary())
		if len(pkt.Payload) > 0 {
			fmt.Print(gonet.FormatHexDump(pkt.Payload))
		}
	}

	fmt.Printf("\n[*] Finished parsing. Total packets: %d\n", count)
}

func runSerial(device string, baud int, mode hw.DelimiterMode) {
	fmt.Printf("[*] Monitoring serial device %s (Baud: %d)...\n", device, baud)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[*] Stopping serial monitor...")
		cancel()
	}()

	cfg := hw.FrameConfig{
		Device:       device,
		BaudRate:     baud,
		Mode:         mode,
		SOF:          0xAA,
		EOF:          0x55,
		MaxFrameSize: 4096,
	}

	monitor := hw.NewSerialMonitor(cfg)
	frameChan, err := monitor.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening serial monitor: %v\n", err)
		os.Exit(1)
	}

	for frame := range frameChan {
		fmt.Println(frame.String())
	}
}
