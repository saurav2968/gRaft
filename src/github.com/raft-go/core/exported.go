package core

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
)

type RaftMode uint

type RaftConfig struct {
	GatewayPort            uint   `yaml:"gatewayPort"`
	ApiPort                uint   `yaml:"apiPort"`
	StateDir               string `yaml:"stateDir"`
	LogLagThreshold        uint   `yaml:"logLagThreshold"`
	LogCompactionThreshold uint   `yaml:"logCompactionThreshold"`
}

func (r RaftConfig) String() string {
	return fmt.Sprintf("RaftConfig{Gateway port: %v, Api port: %v, state_dir: %v, log_lag_threshold: %v, log_compaction_threshold: %v}",
		r.GatewayPort, r.ApiPort, r.StateDir, r.LogLagThreshold, r.LogCompactionThreshold)
}

func NewRaft(configPath string, mode RaftMode) raft {
	// 1. Parse config
	yamlFile, err := ioutil.ReadFile(configPath)
	logrus.Infof("yaml: %s", yamlFile)
	if err != nil {
		log.Fatalf("Unable to read yaml config: %v\n err   #%v ", configPath, err)
	}

	y := RaftConfig{}
	err = yaml.Unmarshal(yamlFile, &y)
	if err != nil {
		log.Fatalf("Unable to parse yaml config.\nerr: %v", err)
	}
	logrus.Infof("Parsed Raft Config as: %s. Raft Mode: %v", y, mode)

	// 2. Validate config and system(e.g port open etc)
	ValidateConfig(y)

	logrus.Infof("Raft Config validated.")

	r := raft{
		config: y,
		mode:   mode,
	}

	//start main go routine
	go r.startGuardian()
	return r
}
