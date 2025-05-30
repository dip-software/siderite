package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dip-software/go-dip-api/iron"
	"github.com/pkg/errors"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "checks system configuration",
		Long:  `check weather your system is configure so it can interact with the HSDP iron`,
		RunE:  doctor,
	}
}

func init() {
	rootCmd.AddCommand(NewDoctorCmd())
}

type proc func() error

var (
	yellow              = color.New(color.FgYellow).SprintFunc()
	green               = color.New(color.FgGreen).SprintFunc()
	red                 = color.New(color.FgRed).SprintFunc()
	pass                = green("[✓]")
	warn                = yellow("[!]")
	problem             = red("[✗]")
	ErrNoClusters       = errors.New("no clusters found in configuration")
	ErrMissingPublicKey = errors.New("missing public key")
)

func testIronCLI() error {
	path, err := exec.LookPath("iron")

	if err != nil {
		fmt.Println(problem, "iron CLI not found. Install it: https://github.com/iron-io/ironcli")
		return err
	}
	out, err := exec.Command(path, "-version").Output()
	if err != nil {
		fmt.Println(problem, "iron CLI failed to run:", err.Error())
		return err
	}
	version := strings.TrimSpace(string(out))
	if version != "0.1.6" {
		fmt.Printf("%s iron CLI version 0.1.6 not detected (version %s)", warn, version)
		return err
	}

	fmt.Printf("%s iron CLI installed (version %s)\n", pass, version)
	return err
}

func testCF() error {
	path, err := exec.LookPath("cf")

	if err != nil {
		fmt.Println(problem, "cf CLI not found. Install it: https://docs.cloudfoundry.org/cf-cli/install-go-cli.html")
		return err
	}
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		fmt.Println(problem, "cf CLI failed to run:", err.Error())
		return err
	}
	version := strings.TrimSpace(string(out))
	fmt.Printf("%s cf CLI installed (%s)\n", pass, version)
	return nil
}

func testConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("%s user home directory issue: %s\n", problem, err.Error())
		return err
	}
	configFile := filepath.Join(home, ".iron.json")
	configJSON, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("%s config file issue: %s\n", problem, err.Error())
		return err
	}
	var jsonConfig iron.Config
	err = json.Unmarshal(configJSON, &jsonConfig)
	if err != nil {
		fmt.Printf("%s error parsing config: %s\n", problem, err.Error())
		return err
	}
	fmt.Printf("%s iron configuration file (%s)\n", pass, configFile)
	if len(jsonConfig.ClusterInfo) == 0 {
		fmt.Printf("%s no clusters found in configuration\n", problem)
		return ErrNoClusters
	}
	fmt.Printf("%s cluster found (%s)\n", pass, jsonConfig.ClusterInfo[0].ClusterID)
	fmt.Printf("%s cluster type (%s)\n", pass, jsonConfig.ClusterInfo[0].ClusterName)
	if jsonConfig.ClusterInfo[0].Pubkey == "" {
		fmt.Printf("%s missing public key for cluster: %s\n", problem,
			jsonConfig.ClusterInfo[0].ClusterID)
		return ErrMissingPublicKey
	}
	fmt.Printf("%s public key for cluster found (%s)\n", pass, jsonConfig.ClusterInfo[0].ClusterID)
	key := jsonConfig.ClusterInfo[0].Pubkey
	_, err = iron.EncryptPayload([]byte(key), []byte("foo"))
	if err != nil {
		fmt.Printf("%s failed to use public key: %s\n", problem, err)
		return err
	}
	fmt.Printf("%s public key is usable\n", pass)
	return nil
}

func doctor(_ *cobra.Command, _ []string) error {
	var errs []error

	e := []proc{
		testIronCLI,
		testConfig,
		testCF,
	}

	for _, p := range e {
		if err := p(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		fmt.Println("some errs were detected")
		return errs[0]
	}
	return nil
}
