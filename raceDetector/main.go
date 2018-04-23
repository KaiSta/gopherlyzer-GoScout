package main

import (
	"flag"

	"./detector"
)

func main() {
	trace := flag.String("trace", "", "path to trace")
	json := flag.Bool("json", false, "output as json")
	plain := flag.Bool("plain", true, "output as plain text")
	bench := flag.Bool("bench", false, "used for benchmarks only")
	analysis := flag.String("mode", "", "select between eraser,racetrack,fasttrack")
	flag.Parse()

	if !*json && !*plain && !*bench {
		panic("no output format defined")
	}

	if trace == nil || *trace == "" {
		panic("no valid trace file")
	}
	if *analysis == "" {
		panic("no analysis mode chosen")
	}

	// if *plain {
	// 	color.HiGreen("Covered schedules")
	// 	color.HiRed("Uncovered schedules")
	// 	color.HiYellow("-----------------------")
	// }

	//	race.Run(*trace, *json, *plain, *bench)
	if *analysis == "fasttrack" {
		race.RunFastTrack(*trace, *json, *plain, *bench)
	} else if *analysis == "racetrack" {
		race.RunRaceTrack(*trace, *json, *plain, *bench)
	} else if *analysis == "eraser" {
		race.RunEraser(*trace, *json, *plain, *bench)
	} else if *analysis == "twintrack" {
		race.RunTwinTrack(*trace, *json, *plain, *bench)
	}

}
