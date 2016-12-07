//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
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

package manager

import (
	"libeopkg"
	"os"
	"path/filepath"
)

// addPackageToRepo will take care of internalising this package into the
// given repository, and exposing the file on the repo filesystem.
func (m *Manager) addPackageToRepo(repo *Repository, pkg *libeopkg.Package) error {
	repoDir := filepath.Join(m.rootDir, repo.GetDirectory())
	if err := os.MkdirAll(repoDir, 00755); err != nil {
		return err
	}
	// TODO: Hardlink the target
	return nil
}

// AddPackage will try to add a single package to the given repo.
func (m *Manager) AddPackage(reponame string, pkgPath string) error {
	repo, err := m.GetRepo(reponame)
	if err != nil {
		return err
	}
	pkg, err := libeopkg.Open(pkgPath)
	if err != nil {
		return err
	}
	defer pkg.Close()
	// Load only the metadata at this point
	if err := pkg.ReadMetadata(); err != nil {
		return err
	}
	// First things first, try to ref the package
	if err := m.pool.RefPackage(pkg); err != nil {
		return err
	}
	if err := m.addPackageToRepo(repo, pkg); err != nil {
		// defer m.pool.UnrefPackage(pkg)
		return err
	}
	return nil
}

// AddPackages will add all of the given packages to the specified resource
func (m *Manager) AddPackages(repoName string, pkgs []string) error {
	// TODO: Check the repo exists!

	// Iterate and open all of the packages
	for _, pkgPath := range pkgs {
		if err := m.AddPackage(repoName, pkgPath); err != nil {
			return err
		}
	}
	return nil
}