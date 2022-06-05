#!/bin/bash

gci write -s Standard -s Default -s "Prefix(github.com/javorszky/battlesnek)" --NoInlineComments --NoPrefixComments main.go $(find pkg/ -type f -name '*.go')
