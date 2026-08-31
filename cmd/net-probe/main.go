package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Ri4ards2006/go-core-lab/internal/hw"
	gonet "github.com/Ri4ards2006/go-core-lab/internal/net"
	"github.com/Ri4ards2006/go-core-lab/pkg/export"
)

func printUsage() {
	fmt.Println("Usage: net-probe [command] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  capture <interface> [-w output.pcap]     Live raw socket packet sniffer")
	fmt.Println("  parse-pcap <file.pcap>                   Parse and replay packets from a PCAP file")
	fmt.Println("  serial <device> [baud] [newline|sync]    Live UART/Serial bus telemetry monitor")
	fmt.Println("  help                                     Show this help message")
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

