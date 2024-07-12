package config_test

import (
	"testing"

	"github.com/CESSProject/cess-dcdn-components/config"
)

func TestParseConfig(t *testing.T) {
	conf := &config.DefaultConfig{}
	err := config.ParseCommonConfig("../config.yaml", "yaml", conf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("parse config success", conf)
}
