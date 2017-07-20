package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jlevesy/envconfig"
)

const (
	AppPrefix = "GROOT"
	Separator = "_"
)

type Lateralizer interface {
	Lateralize() error
}

type SplineReticulator struct {
	Lateralizer
	Red   float64 // GROOT_SPLINERS_<KEY>_RED
	White float32 // GROOT_SPLINERS_<KEY>_WHITE
	Blue  float64 // GROOT_SPLINERS_<KEY>_BLUE
}

type GrootConfig struct {
	LateralizerMode string // GROOT_LATERALIZER_MODE
	Real            bool   // GROOT_REAL
	StreamlingRatio uint64 // GROOT_STREAMLINING_RATIO
	Spliners        map[int]*SplineReticulator
	Timeout         time.Duration // GROOT_TIMEOUT
}

func main() {
	config := &GrootConfig{}

	if err := envconfig.New(AppPrefix, Separator).Load(config); err != nil {
		fmt.Println("Failed to load config, got: ", err)
		os.Exit(1)
	}

	fmt.Println("Loaded AppConfig: ", config)
	fmt.Println("Loaded Spliners: ", config.Spliners)
}
