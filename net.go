package proc

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var procNetFiles = []string{"netstat", "snmp"}

// NetStats parses the files /proc/net/netstat, and /proc/net/snmp.
func NetStats() (map[string]map[string]int64, error) {
	result := make(map[string]map[string]int64)
	for _, filename := range procNetFiles {
		f, err := os.Open("/proc/net/" + filename)
		if err != nil {
			continue
		}
		defer f.Close()
		reader := bufio.NewReader(f)
		keys := []string{}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if len(keys) > 0 {
						return nil, fmt.Errorf("Read a header line without a following data line.")
					}
					break
				}
				return nil, err
			}
			if len(keys) == 0 {
				keys = strings.Fields(line)
			} else {
				values := strings.Fields(line)
				if len(values) != len(keys) {
					return nil, fmt.Errorf("Found a value line of a different length than the header line")
				}
				if keys[0] != values[0] || !strings.HasSuffix(keys[0], ":") {
					return nil, fmt.Errorf("Header or value lines don't match or don't start with a label")
				}
				label := keys[0][:len(keys[0])-1]
				lineMap := make(map[string]int64)
				for i, key := range keys {
					if i == 0 {
						continue
					}
					v, err := strconv.ParseInt(values[i], 10, 64)
					if err != nil {
						return nil, fmt.Errorf("Could not parse value.")
					}
					lineMap[key] = v
				}
				result[label] = lineMap
				keys = nil
			}
		}

		f.Close() // Double close is fine
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("None of the required /proc/net/ files could be found.")
	}
	return result, nil
}
