# First, a container which builds some useful tools; these will then be
# copied into the resulting container
FROM golang:1.19 AS builder

# Creates an app directory to hold your app’s source code
WORKDIR /apps

# Copies everything from your root directory into /app
# should probably have some versioning here...
RUN git clone https://github.com/mozilla/sops.git
RUN git clone https://github.com/FiloSottile/age.git
RUN git clone https://github.com/Mic92/ssh-to-age.git
RUN git clone https://github.com/kevin-hanselman/dud.git 

# build sops
WORKDIR /apps/sops
RUN go mod download
RUN go build go.mozilla.org/sops/v3/cmd/sops
 
# build age
WORKDIR /apps/age
RUN go mod download
RUN go build filippo.io/age/cmd/age
RUN go build filippo.io/age/cmd/age-keygen 

# build age
WORKDIR /apps/ssh-to-age
RUN go mod download
RUN go build github.com/Mic92/ssh-to-age/cmd/ssh-to-age

# build dud
WORKDIR /apps/dud
RUN go mod download
RUN go build github.com/kevin-hanselman/dud

# ---------------------------------------------------------------------------
# This is the start of the resulting container

FROM gltr/base-notebook

COPY --from=builder /apps/sops/sops /usr/local/bin
COPY --from=builder /apps/age/age /usr/local/bin
COPY --from=builder /apps/age/age-keygen /usr/local/bin
COPY --from=builder /apps/ssh-to-age/ssh-to-age /usr/local/bin

USER root
RUN apt-get update && apt-get install -y \
  curl \
  git \
  ssh \
  openssh-server \
  rclone \
  && rm -rf /var/lib/apt/lists/*

# remove the mord - we present content in the bash_profile
RUN rm -fr /etc/update-motd.d/
RUN echo "" > /etc/motd

# the legal notice is removed here, we should include some equivalent or reference
# to it
RUN rm -fr /etc/legal

# install s6-overlay
ADD https://github.com/just-containers/s6-overlay/releases/download/v3.1.2.1/s6-overlay-noarch.tar.xz /tmp
RUN tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz
RUN ARCH=$(uname -m) \
  && curl -sLO https://github.com/just-containers/s6-overlay/releases/download/v3.1.2.1/s6-overlay-$ARCH.tar.xz --output-dir /tmp/ \
  && tar -C / -Jxpf /tmp/s6-overlay-$ARCH.tar.xz

COPY resources/docker/s6-rc.d /etc/s6-overlay/s6-rc.d

# This eanbles the services
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/sshd
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/jupyter-server
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/git-clone
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/ssh-init 
RUN touch /etc/s6-overlay/s6-rc.d/user/contents.d/gltr-init 

# add gltr
ADD gltr /usr/local/bin

# Set up ssh config
COPY resources/docker/sshd-config/sshd_config /etc/ssh/

COPY resources/docker/bash_profile /home/gltr/.bash_profile

# this is for the ssh agent...
RUN mkdir -p /var/run/user/1000
RUN chown 1000:100 /var/run/user/1000
RUN mkdir -p /var/run/user/1000/ssh
RUN chown 1000:100 /var/run/user/1000/ssh

ENTRYPOINT ["/init"]
