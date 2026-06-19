#!/bin/bash
lsof -ti:8080 | xargs -r kill -9