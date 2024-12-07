package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSaveToCloudFlareD1(t *testing.T) {
	entries := []XrayLog{
		{
			User:        "exampleUser",
			IP:          "192.168.1.1",
			Target:      "example.com",
			Inbound:     "inboundTag",
			Outbound:    "outboundTag",
			RequestTime: RequestTime{time.Now()},
		},
	}
	records := map[string]interface{}{
		"records": entries,
	}
	jsonData, _ := json.Marshal(records)
	t.Log(string(jsonData))
}
