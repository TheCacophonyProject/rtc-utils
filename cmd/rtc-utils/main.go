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
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	var args Args
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
		return rtc.Read()
	}

	if args.WriteIfSync {
		sync, err := rtc.IsNTPSynced()
		if err != nil {
			return err
		}
		if sync {
			log.Println("NTP is synchronized. Writing time to RTC")
			return rtc.Write()
		} else {
			log.Println("NTP is not synchronized. Not writing time to RTC")
			return nil
		}
	}

	if args.Write {
		return rtc.Write()
	}

	if args.CheckBattery {
		return rtc.CheckBattery()
	}

	return errors.New("no options given")
}
