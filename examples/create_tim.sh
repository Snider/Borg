#!/bin/bash
# Example of using the 'borg collect' command with the '--format tim' flag.

# This script clones the specified Git repository and saves it as a .tim file.

# Ensure the 'borg' executable is in the current directory or in the system's PATH.
# You can build it by running 'go build' in the project root.
./borg collect github repo https://github.com/Snider/Borg --output borg.tim --format tim
