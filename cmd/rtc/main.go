package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/rjeczalik/notify"

	"github.com/TheCacophonyProject/rtc-utils/rtc"
)

const systemdStatePath = "/var/lib/systemd"

var version = "<not set>"

type ReadCmd struct{}
type CheckBatteryCmd struct{}
type WriteCmd struct {
	Force bool `args:"--force" help:"don't check if NTP is synchronized"`
	Wait  bool `args:"--wait" help:"block until is NTP synchronized before writing"`
}

type Args struct {
	Read         *ReadCmd         `arg:"subcommand:read" help:"read RTC to system time"`
	Write        *WriteCmd        `arg:"subcommand:write" help:"write system time to RTC if NTP is synchronized"`
	CheckBattery *CheckBatteryCmd `arg:"subcommand:check-battery" help:"check if the RTC battery is low"`
	Attempts     int              `args:"--attempts" help:"number of times to try reading/writing registers to the RTC"`
}

func (Args) Version() string {
	return version
}

func main() {
	err := runMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0)

	switch {
	case args.Read != nil:
		return rtc.Read(args.Attempts)
	case args.CheckBattery != nil:
		return rtc.CheckBattery(args.Attempts)
	case args.Write != nil:
		if args.Write.Force {
			log.Println("not checking if NTP is synchronized")
			return rtc.Write(args.Attempts)
		}

		sync, err := checkNTPSync(args.Write.Wait)
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
	default:
		return errors.New("no options given")
	}
}

func procArgs() Args {
	args := Args{
		Attempts: 1,
	}
	p := arg.MustParse(&args)
	if args.Read == nil && args.Write == nil && args.CheckBattery == nil {
		p.Fail("no command given")
	}
	return args
}

func checkNTPSync(wait bool) (bool, error) {
	if wait {
		log.Println("waiting for NTP synchronization")
		err := waitForNTPSync()
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return rtc.IsNTPSynced()
}

func waitForNTPSync() error {
	// Watch the systemd state directory for file writes (specifically
	// the `clock` file). This file is written to by
	// systemd-timesyncd every NTP sync. We watch the directory
	// instead of the file, just in case the file doesn't exist.
	fsEvents := make(chan notify.EventInfo, 1)
	if err := notify.Watch(systemdStatePath, fsEvents, notify.InCloseWrite); err != nil {
		return err
	}
	defer notify.Stop(fsEvents)

	for {
		// Leave if NTP sync has been acheived.
		if sync, err := rtc.IsNTPSynced(); err != nil {
			return err
		} else if sync {
			return nil
		}

		// Wait for filesystem event (hopefully the next NTP sync)
		<-fsEvents
	}
}
