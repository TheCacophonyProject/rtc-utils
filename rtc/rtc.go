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

const (
	rtcAddress        = 0x68
	attemptDelay      = 5 * time.Second
	lowBatteryMessage = "RTC battery is low. Replace soon"
)

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
func Read(attempts int) error {
	state, err := State(attempts)
	if err != nil {
		return err
	}
	if state.LowBattery {
		log.Println(lowBatteryMessage)
	}
	if !state.ClockIntegrity {
		return errors.New("clock integrity is not guaranteed. Update time to ensure proper time")
	}
	timeString := state.Time.Format("2006-01-02T15:04:05")
	log.Printf("time writing to system clock (in UTC): %s", timeString)
	cmd := exec.Command("date", "+%Y-%m-%dT%H:%M:%S", "--utc", fmt.Sprintf("--set=%s", timeString))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running: %s, err: %v, out: %s", cmd.Args, err, string(out))
	}
	return nil
}

func readTime(reg [0x14]byte) (time.Time, error) {
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

	timeString := fmt.Sprintf("20%02d-%02d-%02dT%02d:%02d:%02dZ", year, month, day, hours, minutes, seconds)
	return time.Parse("2006-01-02T15:04:05Z07:00", timeString)
}

type RTCState struct {
	register          [0x14]byte
	Time              time.Time
	LowBattery        bool
	ClockIntegrity    bool
	BatterySwitchOver bool
}

func (s RTCState) String() string {
	return fmt.Sprintf(`RTC State:
	Time (UTC):      %s
	LowBattery:      %t
	Clock Integrity: %t
	Battery Switch:  %t`,
		s.Time.Format("2006-01-02 15:04:05"),
		s.LowBattery,
		s.ClockIntegrity,
		s.BatterySwitchOver)
}

func State(attempts int) (*RTCState, error) {
	reg, err := readRegisters(attempts)
	if err != nil {
		return nil, err
	}
	t, err := readTime(reg)
	if err != nil {
		return nil, err
	}
	return &RTCState{
		register:          reg,
		Time:              t,
		LowBattery:        lowBattery(reg),
		ClockIntegrity:    checkClockIntegrity(reg),
		BatterySwitchOver: reg[2]&byte(1<<3) != 0,
	}, nil
}

func bcdToBin(v uint8) uint8 {
	return v - 6*(v>>4)
}

func binToBCD(v uint8) uint8 {
	return v + 6*(v/10)
}

// Write the current date and set the control registers on the RTC
func Write(attempts int) error {
	reg, err := readRegisters(attempts)
	if err != nil {
		return err
	}
	if lowBattery(reg) {
		log.Println(lowBatteryMessage)
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
	log.Printf("time writing to RTC (in UTC): %s", timeString)

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

	if err := writeRegisters(reg, attempts); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	reg, err = readRegisters(attempts)
	if err != nil {
		return err
	}
	if !checkClockIntegrity(reg) {
		return errors.New("clock integrity was lost after writing time. Likely to be hardware issue")
	}
	return nil
}

func CheckBattery(attempts int) error {
	state, err := State(attempts)
	if err != nil {
		return err
	}

	if state.LowBattery {
		return errors.New(lowBatteryMessage)
	}
	log.Println("RTC battery is fine")
	return nil
}

func readRegisters(attempts int) ([0x14]byte, error) {
	reg, err := readRegistersAttempt()
	if err == nil && reg != [0x14]byte{} {
		return reg, nil
	}
	attempts--
	if attempts <= 0 {
		return [0x14]byte{}, errors.New("failed to read RTC registers")
	}
	log.Printf("failed to read RTC registers. trying %d more times", attempts)
	time.Sleep(attemptDelay)
	return readRegisters(attempts)
}

func checkClockIntegrity(reg [0x14]byte) bool {
	return reg[0x03]&byte(1<<7) == 0
}

func lowBattery(reg [0x14]byte) bool {
	return reg[2]&byte(1<<2) != 0
}

func readRegistersAttempt() (b [0x14]byte, err error) {
	dev, err := getI2CDev()
	if err != nil {
		return
	}
	err = dev.Tx([]byte{0x00}, b[:])
	return b, nil
}

func writeRegisters(reg [0x14]byte, attempts int) error {
	err := writeRegistersAppempt(reg)
	if err == nil {
		return nil
	}
	attempts--
	if attempts <= 0 {
		return errors.New("failed to write RTC registers")
	}
	log.Printf("failed to write RTC registers. trying %d more times", attempts)
	time.Sleep(attemptDelay)
	return writeRegisters(reg, attempts)
}

func writeRegistersAppempt(reg [0x14]byte) error {
	dev, err := getI2CDev()
	if err != nil {
		return err
	}
	return dev.Tx(append([]byte{0x00}, reg[:]...), nil)
}

func IsNTPSynced() (bool, error) {
	out, err := exec.Command("timedatectl").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed check to NTP status: %v - %s", err, string(out))
	}
	syncStrs := [2]string{
		"NTP synchronized: yes",
		"System clock synchronized: yes",
	}
	strOut := string(out)
	for _, syncStr := range syncStrs {
		if strings.Contains(strOut, syncStr) {
			return true, nil
		}
	}
	return false, nil
}
