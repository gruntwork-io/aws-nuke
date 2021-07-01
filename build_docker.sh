#!/usr/bin/env bash
set -euo pipefail

echo "*** Linting this bash script with shellcheck"
docker run --rm -v "$PWD:/mnt" koalaman/shellcheck "$0"
echo "*** Linting with https://github.com/hadolint/hadolint"
docker run --rm -i hadolint/hadolint < Dockerfile
echo "*** Linting with https://github.com/replicatedhq/dockerfilelint"
docker run --rm -v "$PWD/Dockerfile:/Dockerfile" replicated/dockerfilelint /Dockerfile

echo "*** Using CIRCLE_TAG if it is set. If not, use current git commit hash"
VERSION="${CIRCLE_TAG:-$(git rev-parse --short HEAD)}"

echo "*** Building image with VERSION=$VERSION"
DOCKER_BUILDKIT=1 docker build --compress --build-arg "VERSION=$VERSION" --tag "gruntwork-io/cloud-nuke:$VERSION" --tag "gruntwork-io/cloud-nuke:latest" .

echo "*** Smoke-testing the image"
docker run --rm -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_SESSION_TOKEN gruntwork-io/cloud-nuke --version | grep -v "cloud-nuke version $VERSION" && echo "*** TEST FAILED: Version numbers do not match up" && exit 1
echo "*** Smoke-test OK. Build complete"
