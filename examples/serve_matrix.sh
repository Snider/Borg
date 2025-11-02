#!/bin/bash
# Example of using the 'borg serve' command with a .matrix file.

# This script serves the contents of a .matrix file using a static file server.
# The main executable 'borg' is built from the project's root.
# Make sure you have built the project by running 'go build -o borg main.go' in the root directory.

# First, create a .matrix file
./borg collect github repo https://github.com/Snider/Borg --output borg.matrix --format matrix

# Then, serve it
./borg serve borg.matrix --port 9999
