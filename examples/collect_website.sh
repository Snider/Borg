#!/bin/bash
# Example of using the 'borg collect website' command.

# This script crawls the specified website and saves it as a .dat file.
# The main executable 'borg' is built from the project's root.
# Make sure you have built the project by running 'go build -o borg main.go' in the root directory.

./borg collect website https://google.com --output website.dat --depth 1
