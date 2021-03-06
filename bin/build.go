package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type AuroraCLI struct {
	cfg        *BuildConfig
	verbose    bool
	buildDir   string
	configFile string
	o          *log.Logger
}

type BuildConfig struct {
	AppName    string `json:"name"`
	Version    string `json:"version"`
	Public     string `json:"public"`
	Templates  string `json:"templates"`
	Dest       string `json:"dest"`
	Watch      bool   `json:"wtach"`
	Src        string `json:"src"`
	WorkingDir string `json:"-"`
	ConfigDir  string `json:"config"`
	DBDIr      string `json:"database_dir"`
}

func NewCLI() *AuroraCLI {
	return &AuroraCLI{verbose: false, o: log.New(os.Stdout, "", log.Ltime)}
}

func (a *AuroraCLI) Setup() {
	a.log("===SETUP")
	out, err := exec.Command("go", "get", "-v").Output()
	a.logErr(err)
	if len(out) > 0 {
		a.log(fmt.Sprintf("%s \n", out))
	}
	bd := path.Join(a.cfg.WorkingDir, path.Join(a.cfg.Dest, a.cfg.Version))
	a.logErr(os.MkdirAll(bd, 0700))
	a.buildDir = bd
	a.clean()
	a.log("---DONE")
}
func (a *AuroraCLI) RunTests() {
	a.log("===TESTING")
	if a.verbose {
		out, err := exec.Command("go", "test", "-v").Output()
		a.logErr(err)
		if len(out) > 0 {
			a.log(fmt.Sprintf("%s", out))
		}
	} else {
		out, err := exec.Command("go", "test").Output()
		a.logErr(err)
		if len(out) > 0 {
			a.log(fmt.Sprintf("%s", out))
		}
	}
	a.log("---DONE")
}
func (a *AuroraCLI) CreateBinary() {
	a.log("===CREATING BINARY")
	o := filepath.Join(a.cfg.Dest, filepath.Join(a.cfg.Version, a.cfg.AppName))
	src := filepath.Join(a.cfg.Src, a.cfg.AppName+".go")
	out, err := exec.Command("go", "build", "-o", o, "-v", src).Output()
	a.logErr(err)
	if len(out) > 0 {
		a.log(fmt.Sprintf("%s", out))
	}
	a.log("---DONE")
}
func (a *AuroraCLI) Assemble() {
	a.log("===ASSEMBLING")
	// copy public folder
	a.logErr(a.copyDir(a.cfg.Public, path.Join(a.buildDir, a.cfg.Public)))

	//  copy templates
	a.logErr(a.copyDir(a.cfg.Templates, path.Join(a.buildDir, a.cfg.Templates)))

	// copy application configurations
	appCfg := path.Join(a.buildDir, a.cfg.ConfigDir)
	a.logErr(a.copyDir(a.cfg.ConfigDir, appCfg))
	a.log("---DONE")

}
func (a *AuroraCLI) SetupDatabase() {
	a.log("===SETUP DATABASE")

	// create database directory
	if a.cfg.DBDIr == "" {
		a.logErr(os.MkdirAll(path.Join(a.buildDir, "db"), 0700))
	} else if path.IsAbs(a.cfg.DBDIr) {
		a.logErr(os.MkdirAll(a.cfg.DBDIr, 0700))
	} else {
		a.logErr(os.MkdirAll(path.Join(a.buildDir, a.cfg.DBDIr), 0700))
	}
	a.log("---DONE")
}
func (a *AuroraCLI) Build() {
	// load configuration file
	a.logErr(a.loadConfig())

	// setup build env
	a.Setup()

	// run tests
	a.RunTests()

	// create binary
	a.CreateBinary()

	// assemble evrything into the build directory
	a.Assemble()

	// setup database
	a.SetupDatabase()

	a.log("[SUCCESS] build complete.")
}
func (a *AuroraCLI) loadConfig() error {
	a.log("===CONFIGURING BUILD")
	cfg := new(BuildConfig)
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfgFile := filepath.Join(pwd, a.configFile)
	d, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(d, cfg)
	if err != nil {
		return err
	}
	cfg.WorkingDir = pwd
	a.cfg = cfg
	a.log("---DONE")
	return nil
}

func (a *AuroraCLI) log(msg interface{}) {
	if a.verbose {
		a.o.Output(2, fmt.Sprintln(msg))
	}
}
func (a *AuroraCLI) logErr(err error) {
	if err != nil {
		a.o.Output(2, fmt.Sprintln(err))
		os.Exit(1)
	}
}

func (a *AuroraCLI) copyDir(src, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// create dest dir

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(src)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := src + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = a.copyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				break
			}
		} else {
			// perform copy
			err = a.copyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				break
			}
		}

	}
	return
}

func (a *AuroraCLI) copyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}
func (a *AuroraCLI) clean() {
	a.log("===CLEANING")
	a.logErr(os.RemoveAll(a.buildDir))
	a.log("---DONE")
}
func main() {
	v := flag.Bool("v", false, "logs build messages on stdout")
	c := flag.String("c", "config/build.json", "specifies wich configuration file to use")
	flag.Parse()
	a := NewCLI()
	if *v {
		a.verbose = true
	}
	a.configFile = *c
	a.Build()
}
