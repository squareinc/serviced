// Copyright 2014 The Serviced Authors.
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

package user

import (
	"github.com/control-center/serviced/datastore"

	"strings"
)

// NewStore creates a user Store
func NewStore() Store {
	return &userStoreImpl{}
}

// UserStore type for interacting with User persistent storage
type Store interface {
	datastore.EntityStore
}

type userStoreImpl struct {
	datastore.DataStore
}

//Key creates a Key suitable for getting, putting and deleting Users
func Key(id string) datastore.Key {
	id = strings.TrimSpace(id)
	return datastore.NewKey(kind, id)
}

var kind = "user"
