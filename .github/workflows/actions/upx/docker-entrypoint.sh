#!/bin/sh
set -e

# i'm guessing this is the volume-mounted directory where steps: executes in
cd "$GITHUB_WORKSPACE"

if [ ! -z $INPUT_FILE ]; then
  upx $INPUT_UPX_ARGS $INPUT_FILE
fi;
