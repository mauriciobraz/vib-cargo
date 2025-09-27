package main

import (
	"C"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vanilla-os/vib/api"
)

type CargoModule struct {
	Name   string     `json:"name"`
	Type   string     `json:"type"`
	Source api.Source `json:"source"`

	Release   bool `json:"release"`
	NoDefault bool `json:"no-default"`

	InstallPath string   `json:"install-path"`
	Features    []string `json:"features"`
	BuildFlags  []string `json:"build-flags"`
}

func fetchSource(source api.Source, name string, recipe *api.Recipe) error {
	if err := api.DownloadSource(recipe, source, name); err != nil {
		return err
	}
	return api.MoveSource(recipe.DownloadsPath, recipe.SourcesPath, source, name)
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

	if !api.TestArch(module.Source.OnlyArches, C.GoString(arch)) {
		return C.CString("")
	}

	if err := fetchSource(module.Source, module.Name, recipe); err != nil {
		return C.CString(fmt.Sprintf("ERROR: %s", err.Error()))
	}

	workDir := fmt.Sprintf("/sources/%s", api.GetSourcePath(module.Source, module.Name))
	installPath := module.InstallPath
	if installPath == "" {
		installPath = "/usr/bin"
	}

	cargoCmd := "cargo build --verbose"
	if module.Release {
		cargoCmd += " --release"
	}
	if len(module.Features) > 0 {
		cargoCmd += " --features " + strings.Join(module.Features, ",")
	}
	if module.NoDefault {
		cargoCmd += " --no-default-features"
	}
	if len(module.BuildFlags) > 0 {
		cargoCmd += " " + strings.Join(module.BuildFlags, " ")
	}

	binarySubdir := "debug"
	if module.Release {
		binarySubdir = "release"
	}

	binaryPath := fmt.Sprintf("target/%s/%s", binarySubdir, module.Name)

	fullCmd := fmt.Sprintf(
		"cd %s && if ! command -v cargo >/dev/null 2>&1; then echo 'installing cargo...' >&2 && apt-get update && apt-get install -y cargo rustc; fi && %s && cp %s %s/ && chmod +x %s/%s",
		workDir, cargoCmd, binaryPath, installPath, installPath, module.Name,
	)

	return C.CString(fullCmd)
}

func main() {}
