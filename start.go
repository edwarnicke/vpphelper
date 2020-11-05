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

// Package vpphelper provides a simple Start function that will start up a local vpp,
// dial it, and return the grpc.ClientConnInterface
package vpphelper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	govpp "git.fd.io/govpp.git"
	"git.fd.io/govpp.git/core"
	"github.com/edwarnicke/exechelper"
	"github.com/pkg/errors"
	"gopkg.in/fsnotify.v1"

	"github.com/edwarnicke/log"
)

// StartAndDialContext - starts vpp
// Stdout and Stderr for vpp are set to be log.Entry(ctx).Writer().
func StartAndDialContext(ctx context.Context, opts ...Option) (conn *core.Connection, errCh <-chan error) {
	o := &option{
		rootDir:    DefaultRootDir,
		connectCtx: ctx,
	}
	for _, opt := range opts {
		opt(o)
	}

	if err := writeDefaultConfigFiles(ctx, o); err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		return nil, errCh
	}
	logWriter := log.Entry(ctx).WithField("cmd", "vpp").Writer()
	vppErrCh := exechelper.Start("vpp -c "+filepath.Join(o.rootDir, vppConfFilename),
		exechelper.WithContext(ctx),
		exechelper.WithStdout(logWriter),
		exechelper.WithStderr(logWriter),
	)
	select {
	case err := <-vppErrCh:
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		return nil, errCh
	default:
	}

	conn, err := connect(o.connectCtx, filepath.Join(o.rootDir, "/var/run/vpp/api.sock"))
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- err
		close(errCh)
		return nil, errCh
	}

	return conn, vppErrCh
}

func writeDefaultConfigFiles(ctx context.Context, o *option) error {
	configFiles := map[string]string{
		vppConfFilename: fmt.Sprintf(vppConfContents, o.rootDir),
	}
	for filename, contents := range configFiles {
		filename = filepath.Join(o.rootDir, filename)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			log.Entry(ctx).Infof("Configuration file: %q not found, using defaults", filename)
			if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
				return err
			}
			if err := ioutil.WriteFile(filename, []byte(contents), 0600); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(o.rootDir, "/var/run/vpp"), 0700); os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Join(o.rootDir, "/var/log/vpp"), 0700); os.IsNotExist(err) {
		return err
	}
	return nil
}

func connect(ctx context.Context, filename string) (*core.Connection, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	defer func() { _ = watcher.Close() }()
	if err := watcher.Add(filepath.Dir(filename)); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		now := time.Now()
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				if event.Name == filename && event.Op == fsnotify.Create {
					log.Entry(ctx).Infof("%s was created after %s", filename, time.Since(now))
					return govpp.Connect(filename)
				}
				// watch for errors
			case err := <-watcher.Errors:
				return nil, err
			case <-ctx.Done():
				return nil, errors.WithStack(ctx.Err())
			}
		}
	}
	return govpp.Connect(filename)
}
