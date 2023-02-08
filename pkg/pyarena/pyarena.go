/*
 * Copyright 2022 The DIAMBRA Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pyarena

import (
	_ "embed"
	"os/exec"
)

//go:embed check_roms.py
var CheckRoms string

//go:embed list_roms.py
var ListRoms string

//go:embed get_diambra_engine_version.py
var GetDiambraEngineVersion string

func FindPython() string {
	for _, name := range []string{
		"python",
		"python3",
	} {
		_, err := exec.LookPath(name)
		if err == nil {
			return name
		}
	}
	return "python"
}
