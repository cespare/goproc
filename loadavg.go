package proc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
)

func LoadAverages() ([3]float64, error) {
	var avgs [3]float64
	text, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return avgs, err
	}
	fields := bytes.Fields(text)
	if len(fields) < 3 {
		return avgs, fmt.Errorf("found fewer than 3 fields in /proc/loadavg")
	}
	for i := range avgs {
		avgs[i], err = strconv.ParseFloat(string(fields[i]), 64)
		if err != nil {
			return avgs, fmt.Errorf("error parsing /proc/loadavg: %s", err)
		}
	}
	return avgs, nil
}
