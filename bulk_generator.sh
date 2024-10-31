#!/bin/bash

# Loop through student IDs 5839 to 6577
for id in {5839..6577}; do
    echo "Generating certificate for Student ID: $id"
    ./goqr generate-cert -id "$id"
    
    # Check if the command succeeded
    if [ $? -ne 0 ]; then
        echo "Failed to generate certificate for Student ID: $id"
    else
        echo "Certificate generated successfully for Student ID: $id"
    fi
done

