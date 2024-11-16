#!/usr/bin/env sh
# returns a partial ioreg response with HIDIdleTime = 125000111 nanoseconds (0.125 seconds)


cat <<EOF
    | | |   "IOProviderClass" = "IOResources"
    | | |   "IOReportLegendPublic" = Yes
    | | |   "IOProbeScore" = 0
    | | |   "HIDIdleTime" = 125000111
    | | |   "HIDScrollCountIgnoreMomentumScrolls" = Yes
    | | |   "HIDScrollCountAccelerationFactor" = 163840
    | | |   "HIDServiceSupport" = Yes
EOF