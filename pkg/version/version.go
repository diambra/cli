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

package version

import (
	"fmt"
	"runtime/debug"
)

func String() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return Format(info)
}

func Format(info *debug.BuildInfo) string {
	var (
		clean     = false
		revision  = ""
		buildtime = ""
	)
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			buildtime = s.Value
		case "vcs.modified":
			clean = s.Value == "false"
		}
	}
	str := fmt.Sprintf("%s (built: %s, clean: %t, %s)\n\nDependencies:\n", revision, buildtime, clean, info.GoVersion)
	for _, m := range info.Deps {
		str += "- " + FormatModule(m) + "\n"
	}
	return str
	//return fmt.Sprintf("Main: %s\nSettings: %v\nDeps: %v\n", FormatModule(info.Main), info.Settings, info.Deps)

}

func FormatModule(module *debug.Module) string {
	return fmt.Sprintf("%s %s (%s)", module.Path, module.Version, module.Sum)
}

/*
	return fmt.Sprintf("%s %s (branch: %s, revision: %s, built: %s) (%#v %v)",
		Version, runtime.GOOS+"/"+runtime.GOARCH, Branch, Revision, BuildDate, info, ok,
	)
*/
