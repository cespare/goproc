package proc

import (
	"testing"
)

func TestNetStats(t *testing.T) {
	stats, err := NetStats()
	if err != nil {
		t.Fatal(err)
	}
	// Very basic sanity checking
	for _, key := range []string{"Ip", "Tcp", "Udp"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("Expected to find key key %q in net stats.", key)
		}
	}
}
