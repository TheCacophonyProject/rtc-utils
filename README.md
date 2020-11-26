# rtc-utils

Tools for managing the real-time clock in the Cacophony Project
thermal cameras.

## Testing

Testing isn't automated so manually check before PR:
- read command:
  - Can read from RTC with a valid time and low/full battery (check that low battery warning is logged)
  - Won't read from RTC with bad integrity.
  - Won't read when the RTC is not conencted.
- write command:
  - Will write when RTC has a low/full battery only when NTP is synchronized (check that low battery warning is logged)
  - Will write when NTP is not synchronized when using `--force`
  - Won't write when RTC is not connected.
- check-battery command:
  - Low battery <= 2.5V

## Releases

Releases are built using TravisCI. To create a release visit the
[repository on Github](https://github.com/TheCacophonyProject/rtc-utils/releases)
and then follow our [general instructions](https://docs.cacophony.org.nz/home/creating-releases)
for creating a release.

For more about the mechanics of how releases work, see `.travis.yml` and `.goreleaser.yml`.
