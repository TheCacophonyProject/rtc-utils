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
	Read  bool `arg:"--read" help:"read RTC to system time"`
	Write bool `arg:"--write" help:"write system time to RTC"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	var args Args
	arg.MustParse(&args)
	return args
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0)
	if args.Read {
		return rtc.Read()
	}
	if args.Write {
		return rtc.Write()
	}
	return errors.New("no option given")
}
