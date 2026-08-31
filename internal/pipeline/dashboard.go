// ═══════════════════════════════════════════════════════════════════════════
// Package pipeline implements a live ANSI terminal dashboard for real-time telemetry.
// ═══════════════════════════════════════════════════
package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Dashboard renders real-time pipeline performance and telemetry to stdout.
type Dashboard struct {
	engine      *PipelineEngine
	refreshRate time.Duration
}

// NewDashboard creates a dashboard bound to a running pipeline engine.
func NewDashboard(engine *PipelineEngine, refreshRate time.Duration) *Dashboard {
	if refreshRate <= 0 {
		refreshRate = 150 * time.Millisecond
	}
	return &Dashboard{
		engine:      engine,
		refreshRate: refreshRate,
	}
}

// Run blocks and renders the dashboard until the context is canceled.
func (d *Dashboard) Run(ctx context.Context) {
	ticker := time.NewTicker(d.refreshRate)
	defer ticker.Stop()

	// Clear screen on start
	fmt.Print("\033[H\033[2J")

	for {
		select {
		case <-ctx.Done():
			d.Render(true)
			return
		case <-ticker.C:
			d.Render(false)
		}
	}
}

// Render outputs the formatted dashboard state.
func (d *Dashboard) Render(final bool) {
	snap := d.engine.Stats.Snapshot()
	qDepth := d.engine.QueueDepth()
	workers := d.engine.Config.NumWorkers

	// Format bandwidth
	bps := snap.BytesPerSec
	bwStr := fmt.Sprintf("%.2f B/s", bps)
	if bps >= 1024*1024 {
		bwStr = fmt.Sprintf("%.2f MiB/s", bps/(1024*1024))
	} else if bps >= 1024 {
		bwStr = fmt.Sprintf("%.2f KiB/s", bps/1024)
	}

	totBytes := float64(snap.TotalBytes)
	totBytesStr := fmt.Sprintf("%.2f B", totBytes)
	if totBytes >= 1024*1024 {
		totBytesStr = fmt.Sprintf("%.2f MiB", totBytes/(1024*1024))
	} else if totBytes >= 1024 {
		totBytesStr = fmt.Sprintf("%.2f KiB", totBytes/1024)
	}

	out := new(strings.Builder)

	// Move cursor to top-left (avoid full clear flicker)
	out.WriteString("\033[H")

	out.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	out.WriteString("║                 ⚡ GO-CORE-LAB REAL-TIME PIPELINE ENGINE ⚡                 ║\n")
	out.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")

	status := "RUNNING"
	if final {
		status = "STOPPED"
	}
	out.WriteString(fmt.Sprintf("  Status: %-9s  Uptime: %-12s  Workers: %-4d  Queue: %d/%d\n",
		status, snap.Uptime.Round(time.Second), workers, qDepth, d.engine.Config.QueueSize))
	out.WriteString("  ────────────────────────────────────────────────────────────────────────────\n")

	// Throughput & Latency
	out.WriteString(fmt.Sprintf("  Throughput:  %-14s  Bandwidth: %-14s  Total Pkts: %d\n",
		fmt.Sprintf("%.1f pkts/s", snap.PacketsPerSec), bwStr, snap.TotalProcessed))
	out.WriteString(fmt.Sprintf("  Latency:     Avg: %-9s  Min: %-9s  Max: %-9s  Total Vol:  %s\n",
		snap.AvgLatency, snap.MinLatency, snap.MaxLatency, totBytesStr))
	out.WriteString(fmt.Sprintf("  Drops/Errors: Dropped: %-8d Errors: %-8d\n",
		snap.DroppedEvents, snap.ErrorEvents))
	out.WriteString("  ────────────────────────────────────────────────────────────────────────────\n")

	// Protocol Breakdown Matrix
	out.WriteString("  [Protocol Breakdown]\n")
	out.WriteString(fmt.Sprintf("    Layer 3:  IPv4: %-8d  IPv6: %-8d  ARP: %-8d\n",
		snap.IPv4Count, snap.IPv6Count, snap.ARPCount))
	out.WriteString(fmt.Sprintf("    Layer 4:  TCP:  %-8d  UDP:  %-8d  ICMP: %-8d\n",
		snap.TCPCount, snap.UDPCount, snap.ICMPCount))
	out.WriteString(fmt.Sprintf("    Hardware: Serial Frames: %-8d  Other / Raw: %-8d\n",
		snap.SerialCount, snap.OtherCount))
	out.WriteString("  ────────────────────────────────────────────────────────────────────────────\n")

	// Live Event Feed from RingBuffer
	recent := d.engine.RingBuffer.Last(8)
	out.WriteString(fmt.Sprintf("  [Live Stream (Last %d events)]\n", len(recent)))
	if len(recent) == 0 {
		out.WriteString("    (waiting for incoming frames...)\n")
	} else {
		for _, ev := range recent {
			line := ev.Summary()
			if len(line) > 74 {
				line = line[:71] + "..."
			}
			out.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}
	out.WriteString("  ────────────────────────────────────────────────────────────────────────────\n")
	out.WriteString("  Press Ctrl+C to stop pipeline and view summary report.\n")

	fmt.Print(out.String())
}

