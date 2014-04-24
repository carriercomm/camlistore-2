/*
Copyright 2013 The Camlistore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Packaage context provides a Context type to propagate state and cancellation
// information.
package context

import (
	"errors"
	"net/http"
	"sync"
)

// ErrCanceled may be returned by code when it receives from a Context.Done channel.
var ErrCanceled = errors.New("canceled")

// TODO returns a dummy context. It's a signal that the caller code is not yet correct,
// and needs its own context to propagate.
func TODO() *Context {
	return nil
}

// A Context is carries state and cancellation information between calls.
// A nil Context pointer is valid, for now.
type Context struct {
	cancelOnce sync.Once
	done       chan struct{}

	httpClient *http.Client // nil means default
}

// New returns a new Context.
func New() *Context {
	return &Context{
		done: make(chan struct{}),
	}
}

// HTTPClient returns the HTTP Client to use for this context.
func (c *Context) HTTPClient() *http.Client {
	if c != nil {
		if cl := c.httpClient; cl != nil {
			return cl
		}
	}
	return http.DefaultClient
}

// SetHTTPClient sets the HTTP client as returned by HTTPClient.
// SetHTTPClient must not be called concurrently with HTTPClient.
func (c *Context) SetHTTPClient(cl *http.Client) {
	c.httpClient = cl
}

// New returns a child context attached to the receiver parent context c.
// The returned context is done when the parent is done, but the returned child
// context can be canceled indepedently without affecting the parent.
func (c *Context) New() *Context {
	subc := New()
	if c == nil {
		return subc
	}
	go func() {
		<-c.Done()
		subc.Cancel()
	}()
	return subc
}

// Done returns a channel that is closed when the Context is cancelled
// or finished.
func (c *Context) Done() <-chan struct{} {
	if c == nil {
		return nil
	}
	return c.done
}

// IsCanceled reports whether this context has been canceled.
func (c *Context) IsCanceled() bool {
	select {
	case <-c.Done():
		return true
	default:
		return false
	}
}

// Cancel cancels the context. It is idempotent.
func (c *Context) Cancel() {
	if c == nil {
		return
	}
	c.cancelOnce.Do(c.cancel)
}

func (c *Context) cancel() {
	if c.done != nil {
		close(c.done)
	}
}
