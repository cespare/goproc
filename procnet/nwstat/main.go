package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cespare/goproc/procnet"
	termbox "github.com/nsf/termbox-go"
)

var (
	rawKeys = [][2]string{
		{"Tcp", "CurrEstab"},
	}
	deltaKeys = [][2]string{
		{"Tcp", "ActiveOpens"},
		{"Tcp", "PassiveOpens"},
		{"Tcp", "InErrs"},
		{"Udp", "InDatagrams"},
		{"Udp", "OutDatagrams"},
	}

	freqString    = flag.String("freq", "1s", "Poll frequency")
	bucketsString = flag.String("buckets", "2s,10s", "List of bucket sizes to show (must be multiples of freq)")
	freq          time.Duration
	buckets       []time.Duration
	bucketBufs    = make(map[[2]string][]*CircBuf)
)

func init() {
	flag.Parse()
	var err error
	freq, err = time.ParseDuration(*freqString)
	if err != nil {
		fatal(err)
	}
	for _, bucketString := range strings.Split(*bucketsString, ",") {
		bucket, err := time.ParseDuration(bucketString)
		if err != nil {
			fatalf("Error parsing buckets: %s\n", err)
		}
		buckets = append(buckets, bucket)
	}
	if len(buckets) == 0 {
		fatal("Require at least one bucket")
	}
	for _, bucket := range buckets {
		if bucket%freq != 0 {
			fatalf("Bucket size (%v) is not a multiple of frequency (%v)\n", bucket, freq)
		}
		mul := int(bucket / freq)
		for _, key := range deltaKeys {
			// mul+1 is because we want to know mul intervals into the past (need a point at both ends).
			bufs := bucketBufs[key]
			bufs = append(bufs, NewCircBuf(mul+1))
			bucketBufs[key] = bufs
		}
	}
}

func main() {
	if err := termbox.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer termbox.Close()
	termbox.HideCursor()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGABRT)
	ticker := time.NewTicker(freq)
	defer ticker.Stop()

	input := make(chan rune, 10)
	go func() {
		for {
			if event := termbox.PollEvent(); event.Type == termbox.EventKey {
				input <- event.Ch
			}
		}
	}()

	printStats()
	for {
		select {
		case <-ticker.C:
			printStats()
		case <-done:
			return
		case c := <-input:
			if c == 'q' {
				return
			}
		}
	}
}

var (
	curX, curY int
)

func printStats() {
	stats, err := procnet.ReadNetStats()
	if err != nil {
		fatal(err)
	}
	curX = 0
	curY = 0
	if err := termbox.Clear(termbox.ColorDefault, termbox.ColorDefault); err != nil {
		fatal(err)
	}
	for _, key := range rawKeys {
		value := get(stats, key)
		tbPrintf(termbox.ColorWhite, "%s ", name(key))
		tbPrint(termbox.ColorGreen, value)
		tbNewline()
	}
	tbNewline()

	// Delta section header
	tbPrintf(termbox.ColorDefault, "%-20s", "")
	for i, d := range buckets {
		tbPrint(termbox.ColorWhite, centerString(fmt.Sprintf("last %s", d), 15))
		if i < len(buckets)-1 {
			tbPrint(termbox.ColorBlue, " │ ")
		}
	}
	tbNewline()

	for _, key := range deltaKeys {
		value := get(stats, key)
		for _, buf := range bucketBufs[key] {
			buf.Append(value)
		}
		tbFormatBufs(name(key), bucketBufs[key])
		tbNewline()
	}

	tbNewline()
	tbPrint(termbox.ColorWhite, "Press q to quit.")

	if err := termbox.Flush(); err != nil {
		fatal(err)
	}
}

func centerString(s string, size int) string {
	if len(s) >= size {
		return s
	}
	prefix := (size - len(s)) / 2
	buf := &bytes.Buffer{}
	for i := 0; i < prefix; i++ {
		buf.WriteRune(' ')
	}
	buf.WriteString(s)
	for i := 0; i < size-prefix-len(s); i++ {
		buf.WriteRune(' ')
	}
	return buf.String()
}

func tbFormatBufs(name string, bufs []*CircBuf) {
	tbPrintf(termbox.ColorWhite, "%-20s", name)
	for i, d := range buckets {
		value := fmt.Sprintf("%6s", "?")
		perSecond := fmt.Sprintf("%9s", "?")
		circBuf := bufs[i]
		if circBuf.Full() {
			delta := circBuf.Delta()
			deltaPerSecond := float64(delta) / d.Seconds()
			value = fmt.Sprintf("%6d", delta)
			perSecond = fmt.Sprintf("%7.1f/s", deltaPerSecond)
		}
		tbPrintf(termbox.ColorGreen, "%s%s", value, perSecond)
		if i < len(buckets)-1 {
			tbPrint(termbox.ColorBlue, " │ ")
		}
	}
}

func tbPrintf(fg termbox.Attribute, format string, args ...interface{}) {
	tbPrint(fg, fmt.Sprintf(format, args...))
}

func tbPrint(fg termbox.Attribute, args ...interface{}) {
	for _, r := range []rune(fmt.Sprint(args...)) {
		termbox.SetCell(curX, curY, r, fg, termbox.ColorDefault)
		curX++
	}
}

func tbNewline() {
	curY++
	curX = 0
}

func name(key [2]string) string { return key[0] + "." + key[1] }

func get(m map[string]map[string]int64, key [2]string) int64 {
	if m1, ok := m[key[0]]; ok {
		if v, ok := m1[key[1]]; ok {
			return v
		}
	}
	fatal("Cannot find key " + name(key))
	panic("unreached")
}

func fatal(args ...interface{}) {
	termbox.Close()
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	termbox.Close()
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
