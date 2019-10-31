#!/bin/bash

systemctl daemon-reload
systemctl enable rtc
systemctl enable rtc-ntp
