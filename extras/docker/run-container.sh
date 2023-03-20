#! /usr/bin/env bash

CONTAINER_NAME="jn"
CONTAINER_IMAGE="gltr/minimal-notebook"
SSH_PUBLIC_KEY="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCeUnxYGfGebTcsrRHgX0MDyxXv7zxB0coC8bpj8FUntB798fV6z+ZS8AT2jfhlk64RUccs/VSFYYyJyLXSyFsAIJReqEMjBK0Gyq63eMFwiqycvWW5n+tNWH98AlSGurl6aAGhUbL67s12mdF3/xAcyQw/U5hPNxbLuT1g6k0Rn9GCVjA++d76W9xLEYp/rat0Mwp+Rys7/D4Y0x8FLJxl3OqtW+Q5BNDTR0BRV17YtVhchiWwQI063nD/l/3+Vuuw7HvgoKskGTLxEruqoACgq16ld6MFHZMX0HcgyEfMkAXwMC5RhFIA0wS+ueGe5GWPj6UQklEpL1PCh8V5uqVrSNkTkCH8FaK9Z0p4rC43euUalOgm8ujPrvJbiWCoQbtZwR3MpD74pgGBt4HcJlDv2LKwnSo11jB+BUR0zPZO99UBuTDcLtJRfCbKLhnPnuLbRPRD4tEHfFnHjJqG6bRD9hRoQVM0s5TKPQJfH3ZOtmRi6FPXmVabLxfFbxqYQFmalLZHc63UApGUeKogxPTQoRPn1nros13JM5F9CVwe2uTELUWbDLkP8oJHPhd3nDstu88qpdCcYNBcgSrxU36YvtcV3ABIaxUFqFL49rqz9J7dnSI4uXeoLHGlv8qe0k9BeD2G3K0QWRHpWZ7VxMaxfHXIvOM6i7ddWtkCNU7EvQ== cardno:000618245504"
GIT_REPO="https://github.com/ibm-et/jupyter-samples.git"

echo
echo "Stopping container (if running)..."
docker stop ${CONTAINER_NAME}

echo "Running docker container in detached mode..."
docker run --rm -d \
  -e SSH_PUBLIC_KEY="${SSH_PUBLIC_KEY}" \
  -e GIT_REPO="${GIT_REPO}" \
  -p 8888:8888 \
  --name "${CONTAINER_NAME}" \
  ${CONTAINER_IMAGE}

echo
echo
IPADDR=$(docker inspect ${CONTAINER_NAME} | jq -r '.[0].NetworkSettings.Networks.bridge.IPAddress')
echo "Container ${CONTAINER_NAME} running on IP addr ${IPADDR}"

echo
echo "Showing logs (press Ctrl-C to quit - this will not stop the container)..."
echo
docker logs -f "${CONTAINER_NAME}"
