#!/bin/bash

# Prompt the user for the starting and ending Student IDs
read -p "Enter the starting Student ID: " start_id
read -p "Enter the ending Student ID: " end_id

# Define the pause flag file
PAUSE_FILE="pause.flag"

# Loop through the specified range of student IDs
for id in $(seq "$start_id" "$end_id"); do
    echo "Generating certificate for Student ID: $id"
    ./goqr generate-cert -id "$id"
    
    # Check if the command succeeded
    if [ $? -ne 0 ]; then
        echo "Failed to generate certificate for Student ID: $id"
    else
        echo "Certificate generated successfully for Student ID: $id"
    fi

    # Check if the pause flag file exists
    if [ -f "$PAUSE_FILE" ]; then
        echo "Pausing. Remove $PAUSE_FILE to continue."
        while [ -f "$PAUSE_FILE" ]; do
            sleep 1  # Wait until the pause file is removed
        done
        echo "Resuming..."
    fi
done
