package proc

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

// FSTabEntry describes a line from /proc/mounts, which is the fstab format. See 'man fstab' for the meaning
// of these fields.
// The field comments are examples of what might be found there (but see the man page for details).
type FSTabEntry struct {
	Spec    string   // /dev/sda1
	File    string   // /mnt/data
	Vfstype string   // ext4
	Mntops  []string // [rw, relatime]
	Freq    int      // 0
	Passno  int      // 0
}

var mountsErr = errors.New("Cannot parse /proc/mounts")

// Read mount information from /proc/mounts for the current process.
// BUG(caleb): This doesn't handle spaces in mount points, even though fstab specifies an encoding for them.
func Mounts() ([]*FSTabEntry, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	results := []*FSTabEntry{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return nil, mountsErr
		}
		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		passno, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		results = append(results, &FSTabEntry{
			Spec:    fields[0],
			File:    fields[1],
			Vfstype: fields[2],
			Mntops:  strings.Split(fields[3], ","),
			Freq:    freq,
			Passno:  passno,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
