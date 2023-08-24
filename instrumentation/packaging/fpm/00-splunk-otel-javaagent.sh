#!/bin/bash

set -euo pipefail

if [ -f /usr/lib/splunk-instrumentation/instrumentation.conf ]; then
    while read line; do
        key="$(echo "$line" | cut -d '=' -f1)"
        value="$(echo "$line" | cut -d '=' -f2-)"
        case $key in
            java_agent_jar)
                echo "JAVA_TOOL_OPTIONS=-javaagent:${value}"
                ;;
            resource_attributes)
                echo "OTEL_RESOURCE_ATTRIBUTES=${value}"
                ;;
            service_name)
                echo "OTEL_SERVICE_NAME=${value}"
                ;;
            enable_profiler)
                echo "SPLUNK_PROFILER_ENABLED=${value}"
                ;;
            enable_profiler_memory)
                echo "SPLUNK_PROFILER_MEMORY_ENABLED=${value}"
                ;;
            enable_metrics)
                echo "SPLUNK_METRICS_ENABLED=${value}"
                ;;
        esac
    done < /usr/lib/splunk-instrumentation/instrumentation.conf
fi
