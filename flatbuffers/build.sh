#!/bin/bash

cmake -G "Unix Makefiles"
make -j
make
sudo make install