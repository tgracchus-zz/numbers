#!/usr/bin/env bash
source ./scripts/version.sh
rm -rf distributions
mkdir -p distributions
cd cmd/server
for os in linux darwin; do
      echo "building for ${os} ${arch}"
      bynary=../../distributions/numbers-${VERSION}-${os}-x86_64
      CGO_ENABLED=0 GOARCH=amd64 GOOS=${os} go build -o $bynary .
      chmod a+x $bynary
      shasum -a 1 $bynary | awk '{print $1}' > $bynary.sha1sum
      shasum -a 256 $bynary | awk '{print $1}' > $bynary.sha256sum
done

