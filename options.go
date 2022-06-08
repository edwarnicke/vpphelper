// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vpphelper

const (
	// DefaultRootDir - Default value for RootDir
	DefaultRootDir = ""
)

type option struct {
	rootDir          string
	vppConfig        string
	additionalStanza string
}

// Option - Option for use with vppagent.Start(...)
type Option func(opt *option)

// WithRootDir - set a root dir (usually a tmpDir) for all conf files.
func WithRootDir(rootDir string) Option {
	return func(opt *option) {
		opt.rootDir = rootDir
	}
}

// WithVppConfig - vpp.conf template
// %[1]s will be replaced in the template with the value of the rootDir
func WithVppConfig(vppConfig string) Option {
	return func(opt *option) {
		opt.vppConfig = vppConfig
	}
}

// %[2] will be replaced with the stanza
func WithStanza(stanza string) Option {
	return func(opt *option) {
		opt.additionalStanza = stanza
	}
}
