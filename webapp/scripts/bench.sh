#!/bin/bash

curl localhost:8081/initialize
cd ../../benchmarker/; ./benchmarker -t localhost:8081
