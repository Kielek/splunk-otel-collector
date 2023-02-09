// Copyright  Splunk, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows
// +build windows

package subproc

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/signalfx/signalfx-agent/pkg/core/common/constants"
)

// The Windows specific process attributes
func procAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		//Pdeathsig: syscall.SIGTERM,
	}
}

func defaultPythonBinaryExecutable() string {
	return filepath.Join(os.Getenv(constants.BundleDirEnvVar), "python", "python.exe")
}

func defaultPythonBinaryArgs(pkgName string) []string {
	return []string{
		"-u",
		"-m",
		pkgName,
	}
}
