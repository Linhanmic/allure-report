/*----------------------------------------------------------------
 *  Build helper for the Gauge Allure 3 report plugin.
 *----------------------------------------------------------------*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	cgoEnabled        = "CGO_ENABLED"
	goARCH            = "GOARCH"
	goOS              = "GOOS"
	x86               = "386"
	x86_64            = "amd64"
	arm64             = "arm64"
	darwin            = "darwin"
	linux             = "linux"
	windows           = "windows"
	bin               = "bin"
	bundled           = "bundled"
	deploy            = "deploy"
	pluginJSONFile    = "plugin.json"
	pluginID          = "allure-report"
	gauge             = "gauge"
	plugins           = "plugins"
	dotGauge          = ".gauge"
	newDirPermissions = 0o755
)

var deployDir = filepath.Join(deploy, pluginID)

func main() {
	flag.Parse()
	if *install {
		updatePluginInstallPrefix()
		installPlugin(*pluginInstallPrefix)
	} else if *distro {
		createPluginDistro(*allPlatforms)
	} else {
		compile()
	}
}

func compile() {
	if *allPlatforms {
		compileAcrossPlatforms()
		return
	}
	compileGoPackage()
}

func compileGoPackage() {
	runProcess("go", "build", "-o", getGaugeExecutablePath(pluginID))
}

func createPluginDistro(forAllPlatforms bool) {
	if forAllPlatforms {
		for _, platformEnv := range platformEnvs {
			setEnv(platformEnv)
			*binDir = filepath.Join(bin, fmt.Sprintf("%s_%s", platformEnv[goOS], platformEnv[goARCH]))
			fmt.Printf("Creating distro for platform => OS:%s ARCH:%s \n", platformEnv[goOS], platformEnv[goARCH])
			createDistro()
		}
	} else {
		createDistro()
	}
	fmt.Printf("Distributables created in directory => %s \n", deploy)
}

func createDistro() {
	installBundledDeps()
	compileGoPackage()
	packageName := fmt.Sprintf("%s-%s-%s.%s", pluginID, getPluginVersion(), getGOOS(), getArch())
	distroDir := filepath.Join(deploy, packageName)
	copyPluginFiles(distroDir)
	createZipFromUtil(deploy, packageName)
	os.RemoveAll(distroDir)
}

func createZipFromUtil(dir, name string) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if err := os.Chdir(filepath.Join(dir, name)); err != nil {
		panic(err)
	}
	output, err := executeCommand("zip", "-r", filepath.Join("..", name+".zip"), ".")
	fmt.Println(output)
	if err != nil {
		panic(fmt.Sprintf("Failed to zip: %s", err.Error()))
	}
	if err := os.Chdir(wd); err != nil {
		panic(err)
	}
}

func compileAcrossPlatforms() {
	for _, platformEnv := range platformEnvs {
		setEnv(platformEnv)
		fmt.Printf("Compiling for platform => OS:%s ARCH:%s \n", platformEnv[goOS], platformEnv[goARCH])
		compileGoPackage()
	}
}

func installPlugin(installPrefix string) {
	installBundledDeps()
	compileGoPackage()
	copyPluginFiles(deployDir)
	pluginInstallPath := filepath.Join(installPrefix, pluginID, getPluginVersion())
	if err := mirrorDir(deployDir, pluginInstallPath); err != nil {
		panic(fmt.Sprintf("Failed to mirror directory '%s' to '%s': %s", deployDir, pluginInstallPath, err.Error()))
	}
	fmt.Printf("Installed %s to %s\n", pluginID, pluginInstallPath)
}

func copyPluginFiles(destDir string) {
	files := map[string]string{}
	if getGOOS() == windows {
		files[filepath.Join(getBinDir(), pluginID+".exe")] = bin
	} else {
		files[filepath.Join(getBinDir(), pluginID)] = bin
	}
	files[pluginJSONFile] = ""
	files[bundled] = bundled
	copyFiles(files, destDir)
}

func installBundledDeps() {
	bundledDir := bundled
	lockFile := filepath.Join(bundledDir, "package-lock.json")
	if _, err := os.Stat(lockFile); err == nil {
		runProcess("npm", "ci", "--prefix", bundledDir)
		return
	}
	runProcess("npm", "install", "--prefix", bundledDir)
}

func copyFiles(files map[string]string, installDir string) {
	for src, dst := range files {
		base := filepath.Base(src)
		installDst := filepath.Join(installDir, dst)
		fmt.Printf("Copying %s -> %s\n", src, installDst)
		stat, err := os.Stat(src)
		if err != nil {
			panic(err)
		}
		if stat.IsDir() {
			err = mirrorDir(src, installDst)
		} else {
			err = mirrorFile(src, filepath.Join(installDst, base))
		}
		if err != nil {
			panic(err)
		}
	}
}

func mirrorDir(src, dst string) error {
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		suffix, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return mirrorFile(path, filepath.Join(dst, suffix))
	})
}

func mirrorFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), newDirPermissions); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

func getGaugeExecutablePath(file string) string {
	name := file
	if getGOOS() == windows {
		name += ".exe"
	}
	return filepath.Join(getBinDir(), name)
}

func getBinDir() string {
	if *binDir != "" {
		return *binDir
	}
	return filepath.Join(bin, fmt.Sprintf("%s_%s", getGOOS(), getGOARCH()))
}

func getPluginVersion() string {
	pluginProperties, err := getPluginProperties(pluginJSONFile)
	if err != nil {
		panic(err)
	}
	return pluginProperties["version"].(string)
}

func getPluginProperties(jsonPropertiesFile string) (map[string]interface{}, error) {
	data, err := os.ReadFile(jsonPropertiesFile)
	if err != nil {
		return nil, err
	}
	var pluginJSON map[string]interface{}
	if err := json.Unmarshal(data, &pluginJSON); err != nil {
		return nil, err
	}
	return pluginJSON, nil
}

func updatePluginInstallPrefix() {
	if *pluginInstallPrefix != "" {
		return
	}
	if runtime.GOOS == windows {
		prefix := os.Getenv("APPDATA")
		if prefix == "" {
			panic("Failed to find AppData directory")
		}
		*pluginInstallPrefix = filepath.Join(prefix, gauge, plugins)
		return
	}
	home := os.Getenv("HOME")
	if home == "" {
		panic("Failed to find User Home directory")
	}
	*pluginInstallPrefix = filepath.Join(home, dotGauge, plugins)
}

func runProcess(command string, arg ...string) {
	cmd := exec.Command(command, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Execute %v\n", cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func executeCommand(command string, arg ...string) (string, error) {
	cmd := exec.Command(command, arg...)
	bytes, err := cmd.Output()
	return strings.TrimSpace(string(bytes)), err
}

func setEnv(envVariables map[string]string) {
	for k, v := range envVariables {
		os.Setenv(k, v)
	}
}

func getArch() string {
	arch := getGOARCH()
	if arch == x86 {
		return "x86"
	}
	if arch == arm64 {
		return "arm64"
	}
	return "x86_64"
}

func getGOARCH() string {
	if v := os.Getenv(goARCH); v != "" {
		return v
	}
	return runtime.GOARCH
}

func getGOOS() string {
	if v := os.Getenv(goOS); v != "" {
		return v
	}
	return runtime.GOOS
}

var (
	install             = flag.Bool("install", false, "Install to the specified prefix")
	pluginInstallPrefix = flag.String("plugin-prefix", "", "Specifies the prefix where the plugin will be installed")
	distro              = flag.Bool("distro", false, "Creates distributables for the plugin")
	allPlatforms        = flag.Bool("all-platforms", false, "Compiles or creates distributables for all platforms")
	binDir              = flag.String("bin-dir", "", "Specifies OS_PLATFORM specific binaries to install when cross compiling")
	platformEnvs        = []map[string]string{
		{goARCH: arm64, goOS: darwin, cgoEnabled: "0"},
		{goARCH: x86_64, goOS: darwin, cgoEnabled: "0"},
		{goARCH: x86, goOS: linux, cgoEnabled: "0"},
		{goARCH: x86_64, goOS: linux, cgoEnabled: "0"},
		{goARCH: arm64, goOS: linux, cgoEnabled: "0"},
		{goARCH: x86, goOS: windows, cgoEnabled: "0"},
		{goARCH: x86_64, goOS: windows, cgoEnabled: "0"},
	}
)
