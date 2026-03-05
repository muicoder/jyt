package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const version = "v0.3.1"

type Command struct {
	From, To string
}

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout); err != nil {
		must(fmt.Fprintf(os.Stderr, "Error: %v\n", err))
		os.Exit(1)
	}
}
func run(args []string, stdin io.Reader, stdout io.Writer) error {
	var commands = []Command{
		{"json", "yaml"}, {"json", "toml"},
		{"yaml", "json"}, {"yaml", "toml"},
		{"toml", "json"}, {"toml", "yaml"},
	}

	if len(args) < 2 {
		printUsage(stdout, commands)
		return nil
	}

	jyt := args[1]

	if jyt[0] == 45 && (jyt[1] == 86 || jyt[1] == 118 || (jyt[1] == 45 && (jyt[2] == 86 || jyt[2] == 118))) {
		must(fmt.Fprintln(stdout, version))
		return nil
	}

	var cmdMap = make(map[string]Command)

	for _, cmd := range commands {
		aliases := []string{
			cmd.From + "-to-" + cmd.To,
			cmd.From + "2" + cmd.To,
			string(cmd.From[0]) + "2" + string(cmd.To[0]),
			string(cmd.From[0]) + string(cmd.To[0]),
		}
		for _, alias := range aliases {
			cmdMap[alias] = cmd
		}
	}

	cmd, ok := cmdMap[jyt]

	if !ok {
		return fmt.Errorf("unknown command: %s", jyt)
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	result, err := convert(data, cmd.From, cmd.To)
	if err != nil {
		return err
	}
	_, err = stdout.Write(result)
	return err
}
func convert(data []byte, from, to string) ([]byte, error) {
	var v any
	var err error
	switch from {
	case "json":
		err = json.Unmarshal(data, &v)
	case "yaml":
		err = yaml.Unmarshal(data, &v)
	case "toml":
		err = toml.Unmarshal(data, &v)
	default:
		return nil, fmt.Errorf("unsupported source format: %s", from)
	}
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", from, err)
	}
	v = normalize(v)
	if to == "toml" {
		if _, ok := v.(map[string]any); !ok {
			return nil, fmt.Errorf("TOML requires root to be a table, got %T", v)
		}
	}
	switch to {
	case "json":
		data, err = json.MarshalIndent(v, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	case "yaml":
		data, err = yaml.Marshal(v)
	case "toml":
		data, err = toml.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported target format: %s", to)
	}
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", to, err)
	}
	return data, nil
}
func normalize(v any) any {
	switch v := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[fmt.Sprint(k)] = normalize(val)
		}
		return m
	case map[string]any:
		for k, val := range v {
			v[k] = normalize(val)
		}
	case []any:
		for i, val := range v {
			v[i] = normalize(val)
		}
	}
	return v
}
func must(_ int, _ error) {}
func printUsage(w io.Writer, commands []Command) {
	must(fmt.Fprintln(w, "A tridirectional converter between Json, Yaml, and Toml"))
	must(fmt.Fprintln(w, "\nUsage: jyt <COMMAND>"))
	must(fmt.Fprintln(w, "\nCommands:"))
	for _, cmd := range commands {
		desc := fmt.Sprintf("Convert %s to %s", strings.Title(cmd.From), strings.Title(cmd.To))
		aliases := strings.Join([]string{
			cmd.From + "2" + cmd.To,
			string(cmd.From[0]) + "2" + string(cmd.To[0]),
			string(cmd.From[0]) + string(cmd.To[0]),
		}, ", ")
		must(fmt.Fprintf(w, "  %-12s  %s (also as %s)\n", cmd.From+"-to-"+cmd.To, desc, aliases))
	}
}
