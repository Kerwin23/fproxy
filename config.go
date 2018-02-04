package main

import (
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Redis struct {
		Host      string
		Port      int
		Password  string
		Db        int
		MaxIdle   int
		MaxActive int
		Timeout   int
	}
	Scan struct {
		NWorkers int
		Ports    []int `yaml:",flow"`
	}
	Craw struct {
		Template  string
		Task      string
		UserAgent string
		Interval  int
		Distance  int
	}
	Checker struct {
		Annony struct {
			CheckUrl    string
			NWorkers    int
			CheckSize   int
			MaxBodySize int
		}
		History struct {
			NWorkers  int
			CheckUrls []string
			CheckSize int
			UserAgent string
		}
	}
}

func ReadConfig(filepath string) (Config, error) {
	filedata, err := ioutil.ReadFile(filepath)
	if err != nil {
		return Config{}, err
	}
	config := Config{}
	err = yaml.Unmarshal(filedata, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
