//
// Copyright © 2017 Ikey Doherty <ikey@solus-project.com>
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
//

// Package slip provides the Ferry Slip implementation.
//
// This portion of ferryd is responsible for the management of management
// of the repositories, and receives packages from the builders.
// In the ferryd design, packages are ferried to the slip, where it is then
// organised into the repositories.
package slip

import (
	"os"
	"path/filepath"
)

const (
	// DatabasePathComponent is the suffix applied to a working directory
	// for the database file itself.
	DatabasePathComponent = "ferry.db"
)

// The Context is shared between all of the components of ferryd to provide
// working directories and such.
type Context struct {
	BaseDir string // Base directory of operations
	DbPath  string // Path to the main database file
}

// NewContext will construct a context from the given base directory for
// all file path functions
func NewContext(root string) (*Context, error) {
	// Ensure root to context exists
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, err
	}
	// Start with the absolute filepath and then join anything there after
	basedir, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Context{
		BaseDir: basedir,
		DbPath:  filepath.Join(basedir, DatabasePathComponent),
	}, nil
}