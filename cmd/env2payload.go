package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/dip-software/siderite/models"
	"github.com/spf13/cobra"
)

func NewEnv2PayloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env2payload",
		Short: "Converts env output to JSON payload",
		Long: `You can pipe the output of the env command to this command 
which will output a JSON structure with proper escaping`,
		RunE: env2payload,
	}
}

func init() {
	cmd := NewEnv2PayloadCmd()
	rootCmd.AddCommand(cmd)
	cmd.Flags().StringP("include", "i", "", "comma separated list of variables to include")
	cmd.Flags().StringP("exclude", "x", "", "comma separated list of variables to exclude")
	cmd.Flags().StringSliceP("env", "e", []string{}, "add environment variable")
	cmd.Flags().StringSliceP("cmd", "c", []string{}, "command to include")
	cmd.Flags().BoolP("nostdin", "n", false, "skip reading from stdin")
}

var envParse = regexp.MustCompile(`^(.*?)=(.*)$`)

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	return i < len(s) && s[i] == searchterm
}

func env2payload(cmd *cobra.Command, _ []string) error {
	var payload models.Payload

	payload.Version = "1"

	includeList, _ := cmd.Flags().GetString("include")
	excludeList, _ := cmd.Flags().GetString("exclude")
	include := strings.Split(includeList, ",")
	exclude := strings.Split(excludeList, ",")
	sort.Strings(include)
	sort.Strings(exclude)

	if len(include) > 0 && include[0] != "" && len(exclude) > 0 && exclude[0] != "" {
		_, _ = fmt.Fprintf(os.Stderr, "can't use include and exclude simultaneously\n")
		return fmt.Errorf("can't use include and exclude simultaneously")
	}
	envInput := []byte("")
	var err error

	nostdin, _ := cmd.Flags().GetBool("nostdin")

	if !nostdin {
		envInput, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	payload.Env = make(map[string]string)
	var key string

	for _, line := range strings.Split(strings.TrimSuffix(string(envInput), "\n"), "\n") {
		parsed := envParse.FindStringSubmatch(line)
		if len(parsed) == 3 {
			key = parsed[1]
			value := parsed[2]
			if contains(exclude, key) {
				key = ""
				continue
			}
			if len(include) > 0 && include[0] != "" && !contains(include, key) {
				key = ""
				continue
			}
			payload.Env[key] = value
		} else { // Most likely part of the previous ENV
			if len(key) > 0 {
				payload.Env[key] = payload.Env[key] + line
			}
		}
	}
	// Extra environment
	extraVars, _ := cmd.Flags().GetStringSlice("env")
	for _, e := range extraVars {
		parsed := envParse.FindStringSubmatch(e)
		if len(parsed) == 3 {
			key := parsed[1]
			value := parsed[2]
			payload.Env[key] = value
		}
	}
	// Command
	cmdVars, _ := cmd.Flags().GetStringSlice("cmd")
	for _, c := range cmdVars {
		payload.Cmd = append(payload.Cmd, c)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	var out bytes.Buffer
	_ = json.Indent(&out, b, "", "  ")
	_, _ = out.WriteTo(os.Stdout)

	return nil
}
