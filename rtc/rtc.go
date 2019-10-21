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
	"fmt"
	"os/exec"
	"strings"
)

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
