#!/bin/bash

go get github.com/rforberger/demo-operator/api/v1alpha1
make generate
make manifests
make docker-build docker-push IMG=rforberger/demo-operator:latest
