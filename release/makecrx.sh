#!/bin/bash -e
#
# Purpose: Pack a Chromium extension directory into crx format
#
# Based on example at https://developer.chrome.com/extensions/crx#scripts.

set -eu

readonly KEY=$1; shift

# Start in top-level source directory
cd "$(dirname $0)/.."

readonly BUILD_TMP=$(mktemp -d)
trap "rm -rf '${BUILD_TMP}'" EXIT

readonly OUTPUT_DIR="${PWD}/bin"

readonly NAME=$(basename "${PWD}")
readonly CRX="${OUTPUT_DIR}/${NAME}.crx"
readonly PUB="${BUILD_TMP}/${NAME}.pub"
readonly SIG="${BUILD_TMP}/${NAME}.sig"
readonly ZIP="${BUILD_TMP}/${NAME}.zip"

# Zip all output files.
zip -qr -9 -X "${ZIP}" . --include \
	manifest.json \
	\*.html \
	\*.js \
	\*README* \
	\*LICENCE*

# Sign the contents of the zip file.
openssl sha1 -sha1 -binary -sign "${KEY}" < "${ZIP}" > "${SIG}"

# Create public key.
openssl rsa -pubout -outform DER < "${KEY}" > "${PUB}" 2>/dev/null

byte_swap () {
  # Take "abcdefgh" and return it as "ghefcdab"
  echo "${1:6:2}${1:4:2}${1:2:2}${1:0:2}"
}

readonly CRMAGIC_HEX="4372 3234" # Cr24
readonly VERSION_HEX="0200 0000" # 2
readonly PUB_LEN_HEX=$(byte_swap $(printf '%08x\n' $(ls -l "${PUB}" | awk '{print $5}')))
readonly SIG_LEN_HEX=$(byte_swap $(printf '%08x\n' $(ls -l "${SIG}" | awk '{print $5}')))
(
  echo "${CRMAGIC_HEX} ${VERSION_HEX} ${PUB_LEN_HEX} ${SIG_LEN_HEX}" | xxd -r -p
  cat "${PUB}" "${SIG}" "${ZIP}"
) > "${CRX}"

