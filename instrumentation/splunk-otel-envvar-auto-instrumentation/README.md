# Splunk OpenTelemetry Zero Configuration Auto Instrumentation for Linux with Environment Variables

**Splunk OpenTelemetry Zero Configuration Auto Instrumentation for Linux with Environment Variables**
installs and configures Splunk OpenTelemetry Auto Instrumentation agent(s) to automatically instrument supported
applications and services running as `systemd` services or within Bourne-compatible login shells, send the captured
traces and metrics to the [Splunk OpenTelemetry Collector](
https://docs.splunk.com/Observability/gdi/opentelemetry/opentelemetry.html), and then on to [Splunk APM](
https://docs.splunk.com/Observability/apm/intro-to-apm.html).

Currently, the following Auto Instrumentation agents are supported:

- [Java](https://docs.splunk.com/Observability/gdi/get-data-in/application/java/get-started.html)

## Prerequisites/Requirements

- Check agent compatibility and requirements:
  - [Java](https://docs.splunk.com/Observability/gdi/get-data-in/application/java/java-otel-requirements.html)
- [Install and configure](https://docs.splunk.com/Observability/gdi/opentelemetry/install-linux.html) the Splunk
  OpenTelemetry Collector.
- Debian or RPM based Linux distribution (amd64/x86_64 or arm64/aarch64).
- Supported applications and services running as `systemd` services or within Bourne-compatible login shells (`bash`,
  `ksh`, `zsh`, etc).

## Installation

The `splunk-otel-envvar-auto-instrumentation` deb/rpm package provides the following files to enable and configure Auto
Instrumentation agent(s) for `systemd` services and Bourne-compatible login shells:
- [`/etc/profile.d/00-splunk-otel-auto-instrumentation.sh`](#login-shells): Drop-in file with default environment
  variables for Bourne-compatible login shells.
- [`/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf`](#systemd): Drop-in file with default
  environment variables for `systemd` services.
- `/usr/lib/splunk-instrumentation/splunk-otel-javaagent.jar`: The [Splunk OpenTelemetry Auto Instrumentation Java
  Agent](https://docs.splunk.com/Observability/gdi/get-data-in/application/java/splunk-java-otel-distribution.html).
- [`/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties`](#java-configuration-file): The default system
  properties file to [configure the Splunk OpenTelemetry Auto Instrumentation Java Agent](
  https://docs.splunk.com/Observability/gdi/get-data-in/application/java/configuration/advanced-java-otel-configuration.html).

Install this package from [Debian/RPM Package Repositories](
../../docs/getting-started/linux-manual.md#auto-instrumentation-debianrpm-package-repositories), or manually download
and install the individual [Debian/RPM Packages](
../../docs/getting-started/linux-manual.md#auto-instrumentation-debianrpm-packages).

After installation, restart the applicable services or reboot the system to enable the Auto Instrumentation agent(s)
with the default configuration. Optionally, see [Configuration](#configuration) for details about configuring the
installed agent(s).

## Configuration

> Before making any changes, it is recommended to check the configuration of the system or individual services for
> potential conflicts.

- **Java**: See the [Advanced Configuration Guide](
  https://docs.splunk.com/Observability/gdi/get-data-in/application/java/configuration/advanced-java-otel-configuration.html)
  for details about supported options and defaults for the Java agent. These options can be configured via
  environment variables or their corresponding system properties after installation.

  > **Configuration Priority**:
  > 
  > The Java agent can consume configuration options from one or more of the following sources (ordered from highest to
  > lowest priority):
  > 1. Java system properties (`-D` flags) passed directly to the agent. For example,
  >      ```shell
  >      JAVA_TOOL_OPTIONS="-javaagent:/usr/lib/splunk-instrumentation/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
  >      ```
  > 2. [Environment variables](#environment-variables)
  > 3. [Configuration files](#java-configuration-file)

### Environment Variables

#### Login Shells

The default [`/etc/profile.d/00-splunk-otel-auto-instrumentation.sh`](
./packaging/00-splunk-otel-auto-instrumentation.sh) drop-in file defines the following environment variables to
enable/configure the installed agent(s) for Bourne-compatible login shells:
- `JAVA_TOOL_OPTIONS=-javaagent:/usr/lib/splunk-instrumentation/splunk-otel-javaagent.jar`
- `OTEL_JAVAAGENT_CONFIGURATION_FILE=/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties`

Any changes to this file will affect ***all*** login shells, unless overriden by higher-priority system, application, or
shell configurations.

To add/modify/override supported environment variables defined in
`/etc/profile.d/00-splunk-otel-auto-instrumentation.sh` (requires `root` privileges):
1. **Option A**: Update `/etc/profile.d/00-splunk-otel-auto-instrumentation.sh` for the desired environment variables.
   For example:
     ```shell
     $ cat <<EOH > /etc/profile.d/00-splunk-otel-auto-instrumentation.sh
     export JAVA_TOOL_OPTIONS="-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     export OTEL_JAVAAGENT_CONFIGURATION_FILE="/my/custom/splunk-otel-javaagent.properties"
     export SPLUNK_PROFILER_ENABLED="true"
     EOH
     ```
   **Option B**: Create/Modify a higher-priority drop-in file for ***all*** login shells to add or override the
   environment variables defined in `/etc/profile.d/00-splunk-otel-auto-instrumentation.sh`. For example:
     ```shell
     $ cat <<EOH >> /etc/profile.d/99-my-custom-env-vars.sh
     export JAVA_TOOL_OPTIONS="-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     export OTEL_JAVAAGENT_CONFIGURATION_FILE="/my/custom/splunk-otel-javaagent.properties"
     export SPLUNK_PROFILER_ENABLED="true"
     EOH
     ```
   **Option C**: Create/Modify a higher-priority login profile for a ***specific*** user's Bourne-compatible shell to
   add or override the environment variables defined in `/etc/profile.d/00-splunk-otel-auto-instrumentation.sh`. For
   example:
     ```shell
     $ cat <<EOH >> $HOME/.profile
     export JAVA_TOOL_OPTIONS="-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     export OTEL_JAVAAGENT_CONFIGURATION_FILE="/my/custom/splunk-otel-javaagent.properties"
     export SPLUNK_PROFILER_ENABLED="true"
     EOH
     ```
  2. After any configuration changes, reboot the system or log out, log back in, and start the applicable
     services/applications for changes to take effect.

#### Systemd

The default [`/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf`](
./packaging/00-splunk-otel-auto-instrumentation.conf) drop-in file defines the following environment variables to
enable/configure the installed agent(s) for `systemd` services:
- `JAVA_TOOL_OPTIONS=-javaagent:/usr/lib/splunk-instrumentation/splunk-otel-javaagent.jar`
- `OTEL_JAVAAGENT_CONFIGURATION_FILE=/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties`

Any changes to this file will affect ***all*** `systemd` services, unless overriden by higher-priority system or service
configurations.

> ***Note***: `Systemd` supports many options/methods for configuring environment variables at the system level or for
> individual services, and are not limited to the examples below. Consult the documentation specific to your Linux
> distribution or service before making any changes. For general details about `systemd`, see the [`systemd` man page](
> https://www.freedesktop.org/software/systemd/man/index.html).

To add/modify/override supported environment variables defined in
`/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf` (requires `root` privileges):
1. **Option A**: Add/Update `DefaultEnvironment` within
   `/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf` for the desired environment variables. For
   example:
     ```shell
     $ cat <<EOH > /usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf
     [Manager]
     DefaultEnvironment="JAVA_TOOL_OPTIONS=-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     DefaultEnvironment="OTEL_JAVAAGENT_CONFIGURATION_FILE=/my/custom/splunk-otel-javaagent.properties"
     DefaultEnvironment="SPLUNK_PROFILER_ENABLED=true"
     EOH
     ```
   **Option B**: Create/Modify a higher-priority drop-in file for ***all*** services to add or override the environment
   variables defined in `/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf`. For example:
     ```shell
     $ cat <<EOH >> /usr/lib/systemd/system.conf.d/99-my-custom-env-vars.conf
     [Manager]
     DefaultEnvironment="JAVA_TOOL_OPTIONS=-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     DefaultEnvironment="OTEL_JAVAAGENT_CONFIGURATION_FILE=/my/custom/splunk-otel-javaagent.properties"
     DefaultEnvironment="SPLUNK_PROFILER_ENABLED=true"
     EOH
     ```
   **Option C**: Create/Modify a higher-priority drop-in file for a ***specific*** service to add or override the
   environment variables defined in `/usr/lib/systemd/system.conf.d/00-splunk-otel-auto-instrumentation.conf`. For
   example:
     ```shell
     $ cat <<EOH >> /usr/lib/systemd/system/my-service.d/99-my-custom-env-vars.conf
     [Service]
     Environment="JAVA_TOOL_OPTIONS=-javaagent:/my/custom/splunk-otel-javaagent.jar -Dotel.service.name=my-service"
     Environment="OTEL_JAVAAGENT_CONFIGURATION_FILE=/my/custom/splunk-otel-javaagent.properties"
     Environment="SPLUNK_PROFILER_ENABLED=true"
     EOH
     ```
2. After any configuration changes, reboot the system or run the following commands to restart the applicable services
   for the changes to take effect:
     ```shell
     $ systemctl daemon-reload
     $ systemctl restart <service-name>   # replace "<service-name>" and run for each applicable service
     ```

### Configuration Files

#### Java Configuration File

The Java agent is configured by default (via the `OTEL_JAVAAGENT_CONFIGURATION_FILE` [environment variable](
#environment-variables)) to consume system properties from the
[`/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties`](./packaging/splunk-otel-javaagent.properties)
configuration file.

Any changes to this file will affect ***all*** `systemd` services and applications running within login shells, unless
overriden by higher-priority system or service configurations.

To add/modify/override [supported system properties](
https://docs.splunk.com/Observability/gdi/get-data-in/application/java/configuration/advanced-java-otel-configuration.html)
in `/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties` (requires `root` privileges):
1. Update `/usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties` for the desired system properties. For
   example:
     ```shell
     $ cat <<EOH > /usr/lib/splunk-instrumentation/splunk-otel-javaagent.properties
     # This is a comment
     otel.service.name=my-service
     otel.resource.attributes=deployment.environment=my-environment
     splunk.metrics.enabled=true
     splunk.profiler.enabled=true
     splunk.profiler.memory.enabled=true
     EOH
     ```
2. After any configuration changes, reboot the system or restart the applicable services/applications for the changes to
   take effect.

## Uninstall

1. If necessary, back up any drop-in or configuration files listed above.
2. Run the following command to uninstall the `splunk-otel-envvar-auto-instrumentation` package (requires `root`
   privileges):
   - Debian:
       ```shell
       dpkg -P splunk-otel-envvar-auto-instrumentation
       ```
   - RPM:
       ```shell
       rpm -e splunk-otel-envvar-auto-instrumentation
       ```
3. Reboot the system or restart the applicable services/applications for the changes to take effect.
