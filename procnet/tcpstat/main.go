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
)

var (
	currEstabKey     = [2]string{"Tcp", "CurrEstab"}
	activeOpensKey   = [2]string{"Tcp", "ActiveOpens"}
	passiveOpensKey  = [2]string{"Tcp", "PassiveOpens"}
	freqString       = flag.String("freq", "1s", "Poll frequency")
	bucketsString    = flag.String("buckets", "2s,10s", "List of bucket sizes to show (must be multiples of freq)")
	freq             time.Duration
	buckets          []time.Duration
	activeOpensBufs  []*CircBuf
	passiveOpensBufs []*CircBuf
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
		// mul+1 is because we want to know mul intervals into the past (need a point at both ends).
		activeOpensBufs = append(activeOpensBufs, NewCircBuf(mul+1))
		passiveOpensBufs = append(passiveOpensBufs, NewCircBuf(mul+1))
	}
}

func main() {
	fd := os.Stderr.Fd()
	termios, err := MakeRaw(fd)
	if err != nil {
		fatal(err)
	}
	defer Restore(fd, termios)
	defer fmt.Println()

	fmt.Println("Press q to quit")
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGABRT)
	ticker := time.NewTicker(freq)
	defer ticker.Stop()

	input := make(chan byte)
	go func() {
		b := make([]byte, 1)
		for {
			_, err := os.Stdin.Read(b)
			if err == nil {
				input <- b[0]
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
	reset  = "\x1b[m"
	red    = "\033[01;31m"
	green  = "\033[01;32m"
	blue   = "\033[01;34m"

	bullet = fmt.Sprintf("%s⚫%s", red, reset)
)

func printStats() {
	stats, err := procnet.ReadNetStats()
	if err != nil {
		fatal(err)
	}
	buf := &bytes.Buffer{}
	currEstab := get(stats, currEstabKey)
	fmt.Fprintf(buf, "\r%s: %s%d%s  %s  ", name(currEstabKey), green, currEstab, reset, bullet)

	activeOpens := get(stats, activeOpensKey)
	for _, buf := range activeOpensBufs {
		buf.Append(activeOpens)
	}
	formatBuf(buf, name(activeOpensKey), activeOpensBufs)
	fmt.Fprintf(buf, "  %s  ", bullet)

	passiveOpens := get(stats, passiveOpensKey)
	for _, buf := range passiveOpensBufs {
		buf.Append(passiveOpens)
	}
	formatBuf(buf, name(passiveOpensKey), passiveOpensBufs)
	fmt.Fprint(buf, " ")
	os.Stdout.Write(buf.Bytes())
}

func formatBuf(buf *bytes.Buffer, name string, bufs []*CircBuf) {
	fmt.Fprintf(buf, "%s: %s[%s ", name, blue, reset)
	sep := fmt.Sprintf("%s|%s ", blue, reset)
	for i, d := range buckets {
		value := "?"
		circBuf := bufs[i]
		if circBuf.Full() {
			value = fmt.Sprintf("%d", circBuf.Delta())
		}
		fmt.Fprintf(buf, "%s%s%s in %s %s", green, value, reset, d, sep)
	}
	buf.Truncate(buf.Len() - len(sep))
	fmt.Fprintf(buf, "%s]%s", blue, reset)
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
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
