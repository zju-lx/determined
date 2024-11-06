package detect

import (
	"bufio"
    "bytes"
	"os/exec"
	"strings"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/determined-ai/determined/master/pkg/device"
)

// AscendDevice metadata
type AscendDevice struct {
	NPUID int
	ProductName string
	Manufacturer string
	SoftwareVersion string
	SerialNumber string
}

func getAscendVersion() (string, error) {
	cmd := exec.Command("npu-smi", "info", "-t", "board", "-i", "0")
	out, err := cmd.Output()
	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return "", nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing npu-smi to detect GPUs")
		return "", err
	}
	var version string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Software Version") {
			parts := strings.Split(line, ":")
			version = strings.TrimSpace(parts[1])
			break
		}
	}
	return version, nil
}

func getNPUCount() (int, error) {
	cmd := exec.Command("npu-smi", "info", "-l")
	out, err := cmd.Output()
	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return 0, execError
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing npu-smi to detect GPUs")
		return 0, err
	}
	var totalCount int
	scanner := bufio.NewScanner(bytes.NewReader(out))
    if scanner.Scan() {
        firstLine := scanner.Text()
        parts := strings.Split(firstLine, ":")
		totalCount, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, err
		}
	}
	return totalCount, nil
}

func parseNPUInfo(data []byte) (AscendDevice, error) {
	var device AscendDevice
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "NPU ID") {
			for scanner.Scan() {
				line = scanner.Text()
				if strings.TrimSpace(line) == "" {
					break
				}
				parts := strings.Split(line, ":")
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				switch key {
				case "Slot ID":
					var err error
					device.NPUID, err = strconv.Atoi(value)
					if err != nil {
						return device, err
					}
				case "Product Name":
					device.ProductName = value
				case "Manufacturer":
					device.Manufacturer = value
				case "Software Version":
					device.SoftwareVersion = value
				case "Serial Number":
					device.SerialNumber = value
				}
			}
		}
	}
	return device, nil
}

func detectAscendNPUs(visibleGPUs string) ([]device.Device, error) {	
	var deviceIds []int
	if visibleGPUs != "" {
		npuIds := strings.Split(visibleGPUs, ",")
		for _, id := range npuIds {
			deviceId, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			deviceIds = append(deviceIds, deviceId)
		}
	} else {
		npuCount, err := getNPUCount()
		if err != nil {
			return nil, err
		}
		for i := 0; i < npuCount; i++ {
			deviceIds = append(deviceIds, i)
		}
	}

	devices := []AscendDevice{}

	for _, id := range deviceIds {
		cmd := exec.Command("npu-smi", "info", "-t", "board", "-i", strconv.Itoa(id))
		out, err := cmd.Output()
		if err != nil {
			log.WithError(err).WithField("output", string(out)).Warnf(
				"error while executing npu-smi to detect GPUs")
			return nil, err
		}
		device, err := parseNPUInfo(out) 
		devices = append(devices, device)
		if err != nil {
			return nil, err
		}
	}

	result := []device.Device{}

	for _, d := range devices {
		result = append(result, device.Device{
			ID:    device.ID(d.NPUID),
			Brand: d.ProductName,
			UUID:  d.SerialNumber,
			Type:  device.NPU,
		})
	}

	return result, nil
}