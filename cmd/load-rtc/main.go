// rtc-utils: Utilities for managing real-time clocks.
// Copyright (C) 2019  The Cacophony Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const maxAttempts = 10
const attemptInterval = 6 * time.Second
const driverName = "rtc_pcf8523"

var validDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} `)

func main() {
	err := runMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runMain() error {
	if os.Getegid() != 0 {
		return errors.New("run as root")
	}

	remaining := maxAttempts
	for !canReadClock() {
		remaining--
		if remaining < 1 {
			return errors.New("giving up initialising RTC")
		}
		fmt.Printf("Will try %d more times...\n", remaining)

		if err := reloadDriver(); err != nil {
			fmt.Printf("failed to reload driver: %v\n", err)
		}
		time.Sleep(attemptInterval)
	}

	if isNTP, err := isNTPSynced(); err != nil {
		return err
	} else if isNTP {
		fmt.Println("NTP synchronised - syncing system to RTC")
		if err := syncSysToHC(); err != nil {
			return err
		}
	} else {
		fmt.Println("not NTP synchronised - syncing RTC to system")
		if err := syncHCToSys(); err != nil {
			return err
		}
	}

	fmt.Println("Clocks initialised")
	return nil
}

func canReadClock() bool {
	out, err := hwclock("-r")
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Under some conditions hwclock will fail but still return a 0
	// exit code so check for a valid date in the output.
	if !validDate.Match(out) {
		fmt.Printf("failed to read RTC: %s\n", string(out))
		return false
	}

	fmt.Printf("RTC time is: %s\n", strings.TrimSpace(string(out)))

	return true
}

func syncSysToHC() error {
	_, err := hwclock("--systohc")
	return err
}

func syncHCToSys() error {
	_, err := hwclock("--hctosys")
	return err
}

func hwclock(arg string) ([]byte, error) {
	out, err := exec.Command("hwclock", arg).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("hwclock %s: %v - %s", arg, err, string(out))
	}
	return out, nil
}

func reloadDriver() error {
	if out, err := exec.Command("modprobe", "-r", driverName).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unload RTC driver: %v - %s", err, string(out))
	}

	if out, err := exec.Command("modprobe", driverName).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load RTC driver: %v - %s", err, string(out))
	}

	return nil
}

func isNTPSynced() (bool, error) {
	out, err := exec.Command("timedatectl").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed check to NTP status: %v - %s", err, string(out))
	}
	return strings.Contains(string(out), "NTP synchronized: yes"), nil
}
