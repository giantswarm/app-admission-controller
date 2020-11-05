#!/bin/bash

curl -L https://github.com/giantswarm/apptestctl/releases/download/v0.4.1/apptestctl-v0.4.1-linux-amd64.tar.gz > /tmp/apptestctl.tar.gz
cd /tmp || exit
tar xzvf apptestctl.tar.gz
chmod u+x /tmp/apptestctl-v0.4.1-linux-amd64/apptestctl
sudo mv /tmp/apptestctl-v0.4.1-linux-amd64/apptestctl /usr/local/bin

apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)"
