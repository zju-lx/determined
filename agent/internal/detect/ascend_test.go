package detect

import (
	"testing"

)

func TestDetectAscendNPUs(t *testing.T) {
	version, err := getAscendVersion()
	if err == nil {
		t.Logf("Version: %v", version)
	} else {
		t.Errorf("Error: %v", err)
	}
	
	npus, err := detectAscendNPUs("")
	if err == nil {
		for _, npu := range npus {
			t.Logf("NPU: %v", npu)
		}
	} else {
		t.Errorf("Error: %v", err)
	}
}