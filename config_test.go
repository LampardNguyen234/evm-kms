package kms

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLoadConfigFromJsonFile(t *testing.T) {
	filePath := "config-example.json"

	cfg, err := LoadConfigFromJSONFile(filePath)
	if err != nil {
		panic(err)
	}

	jsb, _ := json.MarshalIndent(cfg, "", "\t")
	fmt.Println(string(jsb))
}
