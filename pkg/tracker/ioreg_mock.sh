#!/usr/bin/env sh
# returns a partial ioreg response with HIDIdleTime = 257791791 nanoseconds
cat <<EOF
    | | |   "IOProviderClass" = "IOResources"
    | | |   "IOReportLegendPublic" = Yes
    | | |   "IOProbeScore" = 0
    | | |   "HIDIdleTime" = 257791791
    | | |   "HIDScrollCountIgnoreMomentumScrolls" = Yes
    | | |   "HIDScrollCountAccelerationFactor" = 163840
    | | |   "HIDServiceSupport" = Yes
EOF