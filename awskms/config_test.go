package awskms

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLoadConfigFromFile(t *testing.T) {
	filePath := "./config-example.json"
	cfg, err := LoadConfigFromFile(filePath)
	if err != nil {
		panic(err)
	}
	jsb, _ := json.MarshalIndent(cfg, "", "\t")
	fmt.Println(string(jsb))
}
