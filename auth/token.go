// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
)

const (
	// expirationDelta is a margin of error during which a token should be
	// considered expired. This should help avoid expiration races when server
	// times don't match
	expirationDelta = 10 * time.Second
)

var (
	// TokenFileName is the file in which we store the current token
	TokenFileName = "auth.token"

	currentToken    string
	currentIdentity Identity
	zerotime        time.Time
	expiration      time.Time
	cond            = &sync.Cond{L: &sync.Mutex{}}
)

// TokenFunc is a function that can return an authentication token and its
// expiration time
type TokenFunc func() (string, int64, error)

// RefreshToken gets a new token, sets it as the current, and returns the expiration time
func RefreshToken(f TokenFunc, filename string) (int64, error) {
	log.Debug("Refreshing authentication token")
	token, expires, err := f()
	if err != nil {
		return 0, err
	}
	updateToken(token, time.Unix(expires, 0), filename)
	log.WithField("expiration", expires).Info("Received new authentication token")
	return expires, err
}

// AuthToken returns an unexpired auth token, blocking if necessary until
// authenticated
func AuthToken() string {
	cond.L.Lock()
	defer cond.L.Unlock()
	for expired() {
		cond.Wait()
	}
	return currentToken
}

// A non-blocking call to get an unexpired auth token.  Returns an error
//  If no token exists or if the token is expired
func AuthTokenNonBlocking() (string, error) {
	if currentToken == "" {
		return "", ErrNotAuthenticated
	}

	if expired() {
		return "", ErrIdentityTokenExpired
	}

	return currentToken, nil
}

// CurrentIdentity returns the identity represented by the currently-live token,
// or nil if the token is not yet available
func CurrentIdentity() Identity {
	cond.L.Lock()
	defer cond.L.Unlock()
	return currentIdentity
}

// TokenLoop accepts a function that returns an expiring token. It will then
// periodically refresh that token, one minute before it is due to expire,
// setting the result as the current live token, until the done channel is
// closed.
func TokenLoop(f TokenFunc, tokenfile string, done <-chan interface{}) {
	for {
		expires, err := RefreshToken(f, tokenfile)
		if err != nil {
			log.WithError(err).Warn("Unable to obtain authentication token. Retrying in 10s")
			select {
			case <-done:
				return
			case <-time.After(10 * time.Second):
			}
			continue
		}
		// Reauthenticate 1 minute before the token expires
		expiration := time.Unix(expires, 0).Sub(now())
		refresh := expiration - time.Duration(1*time.Minute)
		select {
		case <-done:
			return
		case <-time.After(refresh):
		}
	}
}

// Watch a token file for changes. Load the token when those changes occur.
func WatchTokenFile(tokenfile string, done <-chan interface{}) error {
	log := log.WithFields(logrus.Fields{
		"tokenfile": tokenfile,
	})

	loadToken := func() {
		data, err := ioutil.ReadFile(tokenfile)
		if err != nil {
			log.WithError(err).Warn("Unable to load authentication token from file. Continuing to watch for changes")
		}
		// No need to handle expires or save file, because we're loading from the file rather
		// than re-requesting authentication tokens
		updateToken(string(data), zerotime, "")
		log.Infof("Updated authentication token from disk")
	}

	// An initial token load without any file changes
	loadToken()

	// Now watch for changes
	filechangechan, err := NotifyOnChange(tokenfile, fsnotify.Write|fsnotify.Create, done)
	if err != nil {
		return err
	}
	for _ = range filechangechan {
		loadToken()
	}
	return nil
}

func now() time.Time {
	return time.Now().UTC()
}

// expired returns whether the currently-live token has expired. If the token
// is empty, is is considered expired for these purposes. If it is the zero
// instant, it never expires. Otherwise, it is expired if the expiration time
// minus a margin of error is in the past.
func expired() bool {
	if currentToken == "" {
		return true
	}
	if expiration.IsZero() {
		return false
	}
	return expiration.Add(-expirationDelta).Before(now())
}

func updateToken(token string, expires time.Time, filename string) {
	WaitForDelegateKeys()
	cond.L.Lock()
	currentToken = token
	currentIdentity = getIdentityFromToken(token)
	expiration = expires
	if filename != "" {
		ioutil.WriteFile(filename, []byte(token), 0600)
	}
	cond.L.Unlock()
	cond.Broadcast()
}

func getIdentityFromToken(token string) Identity {
	identity, err := ParseJWTIdentity(token)
	if err != nil {
		log.WithError(err).Error("Unable to obtain identity from token.")
		return nil
	}
	return identity
}
