#!/bin/bash
set -e

DATA_DIR="/data"
MAP_FILE="$DATA_DIR/nigeria.osrm"
RAW_FILE="$DATA_DIR/nigeria.osm.pbf"

mkdir -p "$DATA_DIR"

if [ ! -f "$MAP_FILE.fileIndex" ]; then
  echo "==> First run: downloading and processing Nigeria map..."

  wget -q --show-progress \
    https://download.geofabrik.de/africa/nigeria-latest.osm.pbf \
    -O "$RAW_FILE"

  echo "==> Extracting..."
  osrm-extract -p /opt/car.lua "$RAW_FILE"

  echo "==> Partitioning..."
  osrm-partition "$MAP_FILE"

  echo "==> Customizing..."
  osrm-customize "$MAP_FILE"

  echo "==> Cleaning up raw file..."
  rm -f "$RAW_FILE"

  echo "==> Map ready."
else
  echo "==> Processed map found, skipping setup."
fi

echo "==> Starting OSRM server..."
exec osrm-routed --algorithm mld --port 5000 "$MAP_FILE"
