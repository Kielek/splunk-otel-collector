#!/bin/bash

# Copyright Splunk Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FPM_DIR="$( cd "$( dirname ${BASH_SOURCE[0]} )" && pwd )"
REPO_DIR="$( cd "$FPM_DIR/../../../" && pwd )"

PKG_NAME="splunk-otel-auto-instrumentation"
PKG_VENDOR="Splunk, Inc."
PKG_MAINTAINER="Splunk, Inc."
PKG_DESCRIPTION="Splunk OpenTelemetry Auto Instrumentation"
PKG_LICENSE="Apache 2.0"
PKG_URL="https://github.com/signalfx/splunk-otel-collector"

INSTALL_DIR="/usr/lib/splunk-instrumentation"
LIBSPLUNK_INSTALL_PATH="${INSTALL_DIR}/libsplunk.so"
JAVA_AGENT_INSTALL_PATH="${INSTALL_DIR}/splunk-otel-javaagent.jar"
CONFIG_INSTALL_PATH="${INSTALL_DIR}/instrumentation.conf"
CONFIG_PATH="$REPO_DIR/instrumentation/install/instrumentation.conf"
SYSTEMD_GENERATOR_PATH="${FPM_DIR}/00-splunk-otel-javaagent.sh"
SYSTEMD_GENERATOR_INSTALL_PATH="${INSTALL_DIR}/systemd-environment-generators/00-splunk-otel-javaagent.sh"

JAVA_AGENT_RELEASE_PATH="${FPM_DIR}/../java-agent-release.txt"
JAVA_AGENT_RELEASE_URL="https://github.com/signalfx/splunk-otel-java/releases/"

PREUNINSTALL_PATH="$FPM_DIR/preuninstall.sh"

get_version() {
    commit_tag="$( git -C "$REPO_DIR" describe --abbrev=0 --tags --exact-match --match 'v[0-9]*' 2>/dev/null || true )"
    if [[ -z "$commit_tag" ]]; then
        latest_tag="$( git -C "$REPO_DIR" describe --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true )"
        if [[ -n "$latest_tag" ]]; then
            echo "${latest_tag}-post"
        else
            echo "0.0.1-post"
        fi
    else
        echo "$commit_tag"
    fi
}

download_java_agent() {
    local tag="$1"
    local dest="$2"
    local dl_url=""
    if [[ "$tag" = "latest" ]]; then
      dl_url="$JAVA_AGENT_RELEASE_URL/latest/download/splunk-otel-javaagent.jar"
    else
      dl_url="$JAVA_AGENT_RELEASE_URL/download/$tag/splunk-otel-javaagent.jar"
    fi

    echo "Downloading $dl_url ..."
    mkdir -p "$( dirname $dest )"
    curl -sfL "$dl_url" -o "$dest"
}

setup_files_and_permissions() {
    local libsplunk="$1"
    local java_agent="$2"
    local buildroot="$3"

    mkdir -p "$buildroot/$(dirname $LIBSPLUNK_INSTALL_PATH)"
    cp -f "$libsplunk" "$buildroot/$LIBSPLUNK_INSTALL_PATH"
    sudo chown root:root "$buildroot/$LIBSPLUNK_INSTALL_PATH"
    sudo chmod 755 "$buildroot/$LIBSPLUNK_INSTALL_PATH"

    mkdir -p "$buildroot/$(dirname $JAVA_AGENT_INSTALL_PATH)"
    cp -f "$java_agent" "$buildroot/$JAVA_AGENT_INSTALL_PATH"
    sudo chown root:root "$buildroot/$JAVA_AGENT_INSTALL_PATH"
    sudo chmod 755 "$buildroot/$JAVA_AGENT_INSTALL_PATH"

    mkdir -p "$buildroot/$(dirname $CONFIG_INSTALL_PATH)"
    cp -f "$CONFIG_PATH" "$buildroot/$CONFIG_INSTALL_PATH"
    sudo chown root:root "$buildroot/$CONFIG_INSTALL_PATH"
    sudo chmod 644 "$buildroot/$CONFIG_INSTALL_PATH"

    mkdir -p "$buildroot/$(dirname $SYSTEMD_GENERATOR_INSTALL_PATH)"
    cp -f "$SYSTEMD_GENERATOR_PATH" "$buildroot/$SYSTEMD_GENERATOR_INSTALL_PATH"
    sudo chown root:root "$buildroot/$SYSTEMD_GENERATOR_INSTALL_PATH"
    sudo chmod 755 "$buildroot/$SYSTEMD_GENERATOR_INSTALL_PATH"
}
