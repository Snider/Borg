#!/bin/bash
# Example of using the 'borg serve' command with a .tim file.

# This script serves the contents of a .tim file using a static file server.

# Ensure the 'borg' executable is in the current directory or in the system's PATH.
# You can build it by running 'go build' in the project root.
# First, create a .tim file
./borg collect github repo https://github.com/Snider/Borg --output borg.tim --format tim

# Now, serve the .tim file
./borg serve borg.tim --port 9999
