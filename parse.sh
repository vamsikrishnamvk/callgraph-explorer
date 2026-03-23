#!/usr/bin/env bash
# Build and run the parser (Mac / Linux)
# Usage: ./parse.sh --repo /path/to/repo --output callgraph.json
#
# All flags are passed directly to the parser binary.
# Run ./parse.sh --help to see all options.

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Build if binary is missing or source is newer
BINARY="$SCRIPT_DIR/parse"
SOURCE="$SCRIPT_DIR/parser/main.go"

if [ ! -f "$BINARY" ] || [ "$SOURCE" -nt "$BINARY" ]; then
  echo "Building parser..."
  cd "$SCRIPT_DIR/parser"
  go build -o "$BINARY" .
  echo "Build complete."
fi

exec "$BINARY" "$@"
