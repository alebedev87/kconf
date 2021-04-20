package main

import (
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultConfigDir             = ".kconf"
	kubeConfigVar                = "KUBECONFIG"
	confPathVar                  = "KCONF_LIBRARY_PATH"
	confDirFileMode  os.FileMode = 0755
)

// Config ...
type Config struct {
	Add    bool
	Set    bool
	List   bool
	Remove bool
	Ops    uint8
}

// Flags ...
func (c *Config) Flags() {
	flag.BoolVar(&c.Add, "a", false, "Add kubeconfig to the library")
	flag.BoolVar(&c.Set, "s", false, "Set current kubeconfig")
	flag.BoolVar(&c.List, "l", false, "List all kubeconfigs from the library")
	flag.BoolVar(&c.Remove, "r", false, "Remove kubeconfig from the library")
	flag.Parse()

	if c.Add {
		c.Ops |= 1
	}
	if c.Set {
		c.Ops |= 2
	}
	if c.List {
		c.Ops |= 4
	}
	if c.Remove {
		c.Ops |= 8
	}

	// default cases

	// no flags specified
	if c.Ops == 0 {
		switch len(os.Args) {
		// no args: list
		case 1:
			c.List = true
		// 1 arg: set
		case 2:
			c.Set = true
		// 2 args: add
		case 3:
			c.Add = true
		}
	}
}

// Args ...
func (c *Config) Args() []string {
	if c.Ops == 0 {
		return os.Args[1:]
	}
	return os.Args[2:]
}

// Validate ...
func (c *Config) Validate() bool {
	switch c.Ops {
	case 0, 1, 2, 4, 8:
		return true
	}
	return false
}

// Handler returns the handler function for the set operation
func (c *Config) Handler() func(string, []string) error {
	if c.Add {
		return addKubeconfig
	}
	if c.Set {
		return setKubeconfig
	}
	if c.List {
		return listKubeconfigs
	}
	if c.Remove {
		return removeKubeconfig
	}

	return func(string, []string) error {
		return nil
	}
}

func addKubeconfig(configPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments")
	}

	var file, slink, symlink string

	switch len(args) {
	case 1:
		slink = strings.Split(path.Base(args[0]), ".")[0]
	case 2:
		fallthrough
	default:
		slink = args[1]
	}
	file = args[0]
	symlink = configPath + "/" + slink

	file, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	if !exists(file) {
		return fmt.Errorf("kubeconfig not found: %s", file)
	}

	if exists(symlink) {
		return fmt.Errorf("kubeconfig already exists: %q", slink)
	}

	if err = os.Symlink(file, symlink); err != nil {
		return err
	}

	fmt.Printf("%s -> %s added\n", slink, file)

	return nil
}

func listSymDir(dir string) ([]fs.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var res []fs.FileInfo
	for _, file := range files {
		if !file.IsDir() && (file.Mode()&fs.ModeSymlink == fs.ModeSymlink) {
			res = append(res, file)
		}
	}
	return res, nil
}

func listKubeconfigs(configPath string, args []string) error {
	files, err := listSymDir(configPath)
	if err != nil {
		return err
	}

	currKubeConfig := os.Getenv(kubeConfigVar)

	var star string
	for i, file := range files {
		if path.Join(configPath, file.Name()) == currKubeConfig {
			star = "* "
		} else {
			star = "  "
		}
		fmt.Printf("%s%d) %s\n", star, i+1, file.Name())
	}
	return nil
}

func makeKubeconfig(configPath string, args []string, result func(string) error) error {
	if len(args) == 0 {
		return fmt.Errorf("not enough arguments")
	}
	files, err := listSymDir(configPath)
	if err != nil {
		return nil
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		// filename not index
		for _, file := range files {
			if file.Name() == strings.TrimSpace(args[0]) {
				return result(path.Join(configPath, file.Name()))
			}
		}
	}

	if idx < 0 || idx > len(files) {
		return fmt.Errorf("index out of range")
	}

	return result(path.Join(configPath, files[idx-1].Name()))
}

func setKubeconfig(configPath string, args []string) error {
	return makeKubeconfig(configPath, args, output)
}

func removeKubeconfig(configPath string, args []string) error {
	return makeKubeconfig(configPath, args, remove)
}

func configPath() (string, error) {
	configPath := strings.TrimSpace(os.Getenv(confPathVar))
	if len(configPath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configPath = homeDir + "/" + defaultConfigDir
	}

	if exists(configPath) {
		return configPath, nil
	}

	return configPath, os.Mkdir(configPath, confDirFileMode)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func output(linkPath string) error {
	fmt.Printf("export %s=%s\n", kubeConfigVar, linkPath)
	return nil
}

func remove(linkPath string) error {
	kubeConfigPath, err := os.Readlink(linkPath)
	if err != nil {
		return err
	}

	if err = os.Remove(linkPath); err != nil {
		return err
	}

	fmt.Printf("%s -> %s removed\n", path.Base(linkPath), kubeConfigPath)
	return nil
}

func main() {
	cfg := &Config{}
	cfg.Flags()

	if !cfg.Validate() {
		fmt.Println("error validating flags")
		os.Exit(1)
	}

	configPath, err := configPath()
	if err != nil {
		fmt.Println("error getting config path:", err)
		os.Exit(1)
	}

	if err = cfg.Handler()(configPath, cfg.Args()); err != nil {
		fmt.Println("error handling request:", err)
		os.Exit(1)
	}
}
