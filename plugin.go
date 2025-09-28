package main

import (
	"C"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vanilla-os/vib/api"
)

type CargoModule struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Sources     []api.Source `json:"sources,omitempty"`
	Source      *api.Source  `json:"source,omitempty"`
	Release     *bool        `json:"release,omitempty"`
	NoDefault   bool         `json:"no-default"`
	InstallPath string       `json:"install-path"`
	Features    []string     `json:"features"`
	BuildFlags  []string     `json:"build-flags"`
}

func (m *CargoModule) getSources() []api.Source {
	if len(m.Sources) > 0 {
		return m.Sources
	}

	if m.Source != nil {
		return []api.Source{*m.Source}
	}

	return []api.Source{}
}

func (m *CargoModule) isRelease() bool {
	if m.Release == nil {
		return true
	}

	return *m.Release
}

func (m *CargoModule) getBuildFlags() []string {
	if len(m.BuildFlags) == 0 {
		return []string{"--locked"}
	}

	return m.BuildFlags
}

func fetchSources(sources []api.Source, name string, recipe *api.Recipe) error {
	for _, src := range sources {
		if err := api.DownloadSource(recipe, src, name); err != nil {
			return err
		}
		if err := api.MoveSource(recipe.DownloadsPath, recipe.SourcesPath, src, name); err != nil {
			return err
		}
	}
	return nil
}

//export PlugInfo
func PlugInfo() *C.char {
	info := &api.PluginInfo{
		Name:             "cargo",
		Type:             api.BuildPlugin,
		UseContainerCmds: false,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return C.CString(fmt.Sprintf("ERROR: %s", err.Error()))
	}
	return C.CString(string(data))
}

//export BuildModule
func BuildModule(moduleInterface *C.char, recipeInterface *C.char, arch *C.char) *C.char {
	var module *CargoModule
	var recipe *api.Recipe

	if err := json.Unmarshal([]byte(C.GoString(moduleInterface)), &module); err != nil {
		return C.CString(fmt.Sprintf("ERROR: %s", err.Error()))
	}

	if err := json.Unmarshal([]byte(C.GoString(recipeInterface)), &recipe); err != nil {
		return C.CString(fmt.Sprintf("ERROR: %s", err.Error()))
	}

	sources := module.getSources()
	if len(sources) == 0 {
		return C.CString("ERROR: No sources specified")
	}

	for _, src := range sources {
		if !api.TestArch(src.OnlyArches, C.GoString(arch)) {
			return C.CString("")
		}
	}

	if err := fetchSources(sources, module.Name, recipe); err != nil {
		return C.CString(fmt.Sprintf("ERROR: %s", err.Error()))
	}

	workDir := fmt.Sprintf("/sources/%s", api.GetSourcePath(sources[0], module.Name))

	cargoCmd := "cargo build"
	if module.isRelease() {
		cargoCmd += " --release"
	}
	if len(module.Features) > 0 {
		cargoCmd += " --features " + strings.Join(module.Features, ",")
	}
	if module.NoDefault {
		cargoCmd += " --no-default-features"
	}
	buildFlags := module.getBuildFlags()
	if len(buildFlags) > 0 {
		cargoCmd += " " + strings.Join(buildFlags, " ")
	}

	fullCmd := fmt.Sprintf(
		`cd %s && export PATH="$HOME/.cargo/bin:$PATH" && if ! command -v cargo >/dev/null 2>&1; then echo 'installing rustup...' >&2 && curl https://sh.rustup.rs -sSf | sh -s -- -y; fi && %s`,
		workDir, cargoCmd,
	)

	return C.CString(fullCmd)
}

func main() {}
