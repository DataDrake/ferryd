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

package libferry

// Response is the base portion for all ferryd responses, and will
// include any relevant information on errors
type Response struct {
	Error       bool   // Whether this response is indication of an error
	ErrorString string // The associated error message
}

// A VersionRequest allows the client to request the current version string
// from the running daemon
type VersionRequest struct {
	Response
	Version string `json:"version"`
}

// An ImportRequest is given to ferryd to ask for the given packages to be
// included into the repository
type ImportRequest struct {
	Response
	Path []string `json:"path"`
}
