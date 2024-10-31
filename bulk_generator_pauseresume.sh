#!/bin/bash

# Define the pause flag file
PAUSE_FILE="pause.flag"

# Toggle the pause file
if [ -f "$PAUSE_FILE" ]; then
    echo "Unpausing the script..."
    rm "$PAUSE_FILE"
else
    echo "Pausing the script..."
    touch "$PAUSE_FILE"
fi
