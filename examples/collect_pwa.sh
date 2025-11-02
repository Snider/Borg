#!/bin/bash
# Example of using the 'borg collect pwa' command.

# This script downloads the specified PWA and saves it as a .dat file.
# The main executable 'borg' is built from the project's root.
# Make sure you have built the project by running 'go build -o borg main.go' in the root directory.

./borg collect pwa --uri https://squoosh.app --output squoosh.dat
