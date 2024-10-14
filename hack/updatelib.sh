#!/bin/sh
# Set k8s api version

K8SVERSION=1.30.4
LIBS="client-go api apimachinery"

for lib in ${LIBS}
do
	go get -u k8s.io/${lib}@kubernetes-${K8SVERSION}
done
