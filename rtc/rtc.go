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

package rtc

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"
)

const rtcAddress = 0x68

type rtcRegisters [0x13]byte

func getI2CDev() (*i2c.Dev, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}
	bus, err := i2creg.Open("")
	if err != nil {
		return nil, err
	}

	return &i2c.Dev{Bus: bus, Addr: rtcAddress}, nil
}

// Read will set system time from RTC.
func Read() error {
	reg, err := readRegisters()
	if err != nil {
		return err
	}

	// lowBattery := reg[2]&byte(1<<2) != 0 // TODO send a warning if this is true
	// batterySwitchOver := reg[2]&byte(1<<3) != 0 Might be usefull?

	clockIntegrity := reg[0x03]&byte(1<<7) == 0 // If the clock integrity can be guaranteed
	if !clockIntegrity {
		return errors.New("clock integrity is not guaranteed. Update time to ensure proper time")
	}

	seconds := bcdToBin(reg[0x03] & byte(0x7f))
	minutes := bcdToBin(reg[0x04] & byte(0x7f))
	hour24Format := reg[0x00]&byte(1<<3) == 0
	var hours uint8
	if hour24Format {
		hours = bcdToBin(reg[0x05] & byte(0x3f))
	} else {
		hours = bcdToBin(reg[0x05] & byte(0x1f))
		if reg[0x05]&byte(1<<5) != 0 {
			hours += 12 // in the PM
		}
	}
	day := bcdToBin(reg[0x06] & byte(0x7f))
	month := bcdToBin(reg[0x08] & byte(0x7f))
	year := bcdToBin(reg[0x09] & byte(0x7f))

	timeString := fmt.Sprintf("%02d-%02d-%02dT%02d:%02d:%02d", year, month, day, hours, minutes, seconds)
	log.Println(timeString)
	cmd := exec.Command("date", "+%y-%m-%dT%H:%M:%S", "--utc", fmt.Sprintf("--set=%s", timeString))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running command. command: %s, err: %v, out: %s", cmd.Args, err, string(out))
	}
	return nil
	/*
		reg[0x00] = read[0x00] & ^byte(1<<7) // CAP_SEL to 0 for setting crystal load capacitance to 7pF
		//read[0x01] = read[0x01]
		//read[0x02] = read[0x02]
		read[0x0A] = 0x80 // disable minute alarm
		read[0x0B] = 0x80 // disable hour alarm
		read[0x0C] = 0x80 // disable day alarm
		read[0x0D] = 0x80 // disable week day alarm
		read[0x0F] = 0x38 // disable CLKOUT

		// write controll registers
		if err := dev.Tx(append([]byte{0x00}, read[:3]...), nil); err != nil {
			return err
		}

		// write alarm and clkout registers
		if err := dev.Tx(append([]byte{0x0A}, read[0x0A:0x10]...), nil); err != nil {
			return err
		}
	*/
}

func bcdToBin(v uint8) uint8 {
	return v - 6*(v>>4)
}

func binToBCD(v uint8) uint8 {
	return v + 6*(v/10)
}

// Write the current date and set the control registers on the RTC
func Write() error {
	reg, err := readRegisters()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	second := now.Second()
	minute := now.Minute()
	hour := now.Hour()
	day := now.Day()
	dayOfWeek := now.Weekday()
	month := now.Month()
	year := now.Year()
	timeString := fmt.Sprintf("%02d-%02d-%02dT%02d:%02d:%02d", year, month, day, hour, minute, second)
	log.Println(timeString)

	// https://www.nxp.com/docs/en/data-sheet/PCF8523.pdf
	reg[0x00] = reg[0x00] & ^byte(1<<7) // CAP_SEL to 0 for setting crystal load capacitance to 7pF.
	reg[0x00] = reg[0x00] & ^byte(1<<3) // Set to 24 hour mode
	reg[0x00] = reg[0x00] & ^byte(7<<0) // Disable second, alarm, and correction cycle interrupts

	reg[0x01] = reg[0x01] & ^byte(7<<0) // Disable watchdog timer A, countdown timer A, and countdown timer B

	reg[0x02] = reg[0x02] & ^byte(7<<5) // Enable battery switch over and battery low detection.

	reg[0x03] = binToBCD(uint8(second))
	reg[0x04] = binToBCD(uint8(minute))
	reg[0x05] = binToBCD(uint8(hour))
	reg[0x06] = binToBCD(uint8(day))
	reg[0x07] = uint8(dayOfWeek)
	reg[0x08] = binToBCD(uint8(month))
	reg[0x08] = binToBCD(uint8(month))
	reg[0x09] = binToBCD(uint8(year % 100))

	reg[0x0A] = 0x80 // disable minute alarm
	reg[0x0B] = 0x80 // disable hour alarm
	reg[0x0C] = 0x80 // disable day alarm
	reg[0x0D] = 0x80 // disable week day alarm
	reg[0x0E] = 0x00 // set offset to 0
	reg[0x0F] = 0x38 // disable CLKOUT and timers
	reg[0x10] = 0x07 // Set timer A frequency to 1/3600 Hz (lowest option)
	reg[0x12] = 0x07 // Set timer B frequency to 1/3600 Hz (lowest option)

	dev, err := getI2CDev()
	if err != nil {
		return err
	}

	return dev.Tx(append([]byte{0x00}, reg[:]...), nil)
}

// return all the registers from the RTC
func readRegisters() (b [0x14]byte, err error) {
	dev, err := getI2CDev()
	if err != nil {
		return
	}
	err = dev.Tx([]byte{0x00}, b[:])
	log.Printf("RTC registers: %v", b)
	return b, nil
}

func SyncSysToHC() error {
	_, err := Hwclock("--systohc")
	return err
}

func SyncHCToSys() error {
	_, err := Hwclock("--hctosys")
	return err
}

func Hwclock(arg string) ([]byte, error) {
	out, err := exec.Command("hwclock", arg).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("hwclock %s: %v - %s", arg, err, string(out))
	}
	return out, nil
}

func IsNTPSynced() (bool, error) {
	out, err := exec.Command("timedatectl").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed check to NTP status: %v - %s", err, string(out))
	}
	return strings.Contains(string(out), "NTP synchronized: yes"), nil
}
