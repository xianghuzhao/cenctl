#!/bin/sh

#go-bindata -pkg icon -debug -o assets.go *.ico
go-bindata -pkg icon -o assets.go *.ico
