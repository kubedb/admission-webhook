#!/usr/bin/env bash

pushd $GOPATH/src/github.com/kubedb/apiserver/hack/gendocs
go run main.go
popd
