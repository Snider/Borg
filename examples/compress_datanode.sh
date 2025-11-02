#!/bin/bash
# Example of using the 'borg collect' command with the '--compression' flag.

# This script clones the specified Git repository and saves it as a compressed .dat file.
# The main executable 'borg' is built from the project's root.
# Make sure you have built the project by running 'go build -o borg main.go' in the root directory.

./borg collect github repo https://github.com/Snider/Borg --compression gz
