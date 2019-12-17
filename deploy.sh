#!/bin/sh

export GPG_TTY=$(tty)

set -e

# init key for pass
echo pass | gpg2 --batch --gen-key <<-EOF ; pass init $(gpg --no-auto-check-trustdb --list-secret-keys | grep ^sec | cut -d/ -f2 | cut -d" " -f1)
%echo Generating a standard key
Key-Type: DSA
Key-Length: 1024
Subkey-Type: ELG-E
Subkey-Length: 1024
Name-Real: Meshuggah Rocks
Name-Email: meshuggah@example.com
Expire-Date: 0
# Do a commit here, so that we can later print "done" :-)
%commit
%echo done
EOF

echo "$DOCKER_PASSWORD" | docker login docker.pkg.github.com --username "$DOCKER_USERNAME" --password-stdin

if [ "$(command -v docker-credential-pass)" = "" ]; then
  docker run --rm -itv sh -c "cp /go/bin/docker-credential-pass /src"
fi