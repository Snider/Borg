#!/bin/bash
# Example of using the 'borg collect github repo' command.

# This script clones the specified Git repository and saves it as a .dat file.
# The main executable 'borg' is built from the project's root.
# Make sure you have built the project by running 'go build -o borg main.go' in the root directory.

./borg collect github repo https://github.com/Snider/Borg --output borg.dat
