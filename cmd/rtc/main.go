package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/TheCacophonyProject/rtc-utils/rtc"
	"github.com/alexflint/go-arg"
)

func main() {
	err := runMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var version = "<not set>"

type Args struct {
	Read         bool `arg:"--read" help:"read RTC to system time"`
	WriteIfSync  bool `arg:"--write-if-sync" help:"write system time to RTC if NTP is synchronized"`
	Write        bool `arg:"--write" help:"write system time to RTC"`
	CheckBattery bool `arg:"--check-battery" help:"check if the RTC battery is low"`
	Attempts     int  `args:"--attempts" help:"number of times to try reading/writing registers to the RTC"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	args := Args{
		Attempts: 1,
	}
	p := arg.MustParse(&args)
	if !args.CheckBattery && !args.Read && !args.Write && !args.WriteIfSync {
		p.Fail("no options given")
	}
	return args
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0)

	if args.Read {
		return rtc.Read(args.Attempts)
	}

	if args.WriteIfSync {
		sync, err := rtc.IsNTPSynced()
		if err != nil {
			return err
		}
		if sync {
			log.Println("NTP is synchronized. Writing time to RTC")
			return rtc.Write(args.Attempts)
		} else {
			log.Println("NTP is not synchronized. Not writing time to RTC")
			return nil
		}
	}

	if args.Write {
		return rtc.Write(args.Attempts)
	}

	if args.CheckBattery {
		return rtc.CheckBattery(args.Attempts)
	}

	return errors.New("no options given")
}
