#!/bin/bash

# Example of how to use the 'collect git' command.

# This will clone a single git repository and store it in a DataNode.
borg collect git --uri https://github.com/torvalds/linux.git --output linux.dat

# This will clone all public repositories for a user and store them in a directory.
borg collect git --user torvalds --output /tmp/borg-repos
