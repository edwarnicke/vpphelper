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

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"git.fd.io/govpp.git"
	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core"
	"github.com/edwarnicke/log"
	"github.com/pkg/errors"
	"gopkg.in/fsnotify.v1"
)

type connection struct {
	*core.Connection
	ready chan struct{}
	err   error
}

func newConnection(ctx context.Context, filename string) Connection {
	c := &connection{
		ready: make(chan struct{}),
	}
	go func(ctx context.Context, filename string, c *connection) {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			c.err = err
			return
		}
		defer func() { _ = watcher.Close() }()
		if err = watcher.Add(filepath.Dir(filename)); err != nil {
			c.err = errors.WithStack(err)
			return
		}
		now := time.Now()
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			for {
				select {
				// watch for events
				case event := <-watcher.Events:
					if event.Name == filename && event.Op == fsnotify.Create {
						log.Entry(ctx).Infof("%s was created after %s", filename, time.Since(now))
						c.Connection, c.err = govpp.Connect(filename)
						close(c.ready)
						return
					}
					// watch for errors
				case err = <-watcher.Errors:
					c.err = errors.WithStack(err)
					return
				case <-ctx.Done():
					c.err = errors.WithStack(ctx.Err())
					return
				}
			}
		}
		if err != nil {
			c.err = errors.WithStack(err)
			return
		}
		log.Entry(ctx).Infof("%s was created after %s", filename, time.Since(now))
		c.Connection, c.err = govpp.Connect(filename)
		close(c.ready)
	}(ctx, filename, c)
	return c
}

func (c *connection) NewStream(ctx context.Context) (api.Stream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ready:
		if c.err != nil {
			return nil, c.err
		}
	}
	return c.Connection.NewStream(ctx)
}

func (c *connection) Invoke(ctx context.Context, req, reply api.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ready:
		if c.err != nil {
			return c.err
		}
	}
	return c.Connection.Invoke(ctx, req, reply)
}

func (c *connection) NewAPIChannel() (api.Channel, error) {
	<-c.ready
	if c.err != nil {
		return nil, c.err
	}
	return c.Connection.NewAPIChannel()
}

func (c *connection) NewAPIChannelBuffered(reqChanBufSize, replyChanBufSize int) (api.Channel, error) {
	<-c.ready
	if c.err != nil {
		return nil, c.err
	}
	return c.Connection.NewAPIChannelBuffered(reqChanBufSize, replyChanBufSize)
}

var _ Connection = &connection{}
