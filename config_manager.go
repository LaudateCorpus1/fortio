package fortio

import (
	"fmt"
	"regexp"

	"unicode"

	"os"
	"reflect"
	"strconv"
	"strings"

	"log"

	"bytes"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	tagName = "config"

	environmentVariable namespace = "env"
	configURL           namespace = "url"
)

var camelCaseRegex = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

// namespace identifies the which namespace the config belongs to like
// environment variable, cassandra registry or kafka registry, so to
// parse accordingly
type namespace string

// Manager auto wires the given config pointer with given set of
// config loaders to NewConfigManager API
type Manager struct {
	appName       string
	rootCmd       *cobra.Command
	logger        Logger
	configLoaders []ConfigLoader
}

// NewConfigManagerWithRootCmd returns a configManager using the provided rootCmd
func NewConfigManagerWithRootCmd(rootCmd *cobra.Command, configLoaders ...ConfigLoader) *Manager {
	return &Manager{
		rootCmd:       rootCmd,
		configLoaders: configLoaders,
	}
}

// NewConfigManager returns new instance of ConfigManager
func NewConfigManager(appName, description string, configLoaders ...ConfigLoader) *Manager {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: description,
	}

	helpCmd := &cobra.Command{
		Use:   "help",
		Short: fmt.Sprintf("help for %s", appName),
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.Flags().BoolP("help", "h", true, fmt.Sprintf("help for %s", appName))
			cmd.Usage()
		},
	}

	rootCmd.SetHelpCommand(helpCmd)
	rootCmd.AddCommand(helpCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Version of Config Manager",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("1.0")
			os.Exit(0)
		},
	}

	rootCmd.AddCommand(versionCmd)

	return &Manager{
		appName:       appName,
		logger:        NewStdLogger(3, log.Ldate|log.Ltime),
		rootCmd:       rootCmd,
		configLoaders: append(configLoaders, &CmdLineConfigLoader{}),
	}
}

// SetLogger will set given logger and uses it for logging
func (cm *Manager) SetLogger(logger Logger) {
	cm.logger = logger
}

// Load will create command line flags for given config and loads values into
// it from environment variables
func (cm *Manager) Load(config Config) error {
	return cm.load(config, true)
}

func (cm *Manager) load(config Config, execute bool) error {
	err := cm.createCommandLineFlags(cm.rootCmd, config)
	if err != nil {
		cm.logger.Errorf("Unable to load config - %v", err)
		return err
	}

	cm.rootCmd.Run = func(cmd *cobra.Command, args []string) {}

	if execute {
		if err := cm.rootCmd.Execute(); err != nil {
			cm.logger.Debugf("Command line args: %+v", os.Args)
			cm.logger.Errorf("Error executing rootCmd - %v", err)
			return err
		}
	}

	for _, loader := range cm.configLoaders {
		err := loader.Load(config)
		if err != nil {
			return err
		}
	}

	helpIsSet, _ := cm.rootCmd.Flags().GetBool("help")
	if helpIsSet {
		os.Exit(0)
	}

	return nil
}

// createCommandLineFlags will create command line flags for given config via Cobra and Viper
// to support command line overriding of config values
func (cm *Manager) createCommandLineFlags(cmd *cobra.Command, config interface{}) error {
	fields := make(map[string]field)
	getAllFields(config, fields)
	for name, field := range fields {
		lFirst := lowerFirst(name)

		switch ptr := field.addr.(type) {
		case *string:
			if field.defaultValue != "" {
				viper.SetDefault(lFirst, field.defaultValue)
			}
			cmd.PersistentFlags().String(lFirst, *ptr, field.usage)
		case *int:
			val, err := strconv.Atoi(field.defaultValue)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a int", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Int(lFirst, *ptr, field.usage)
		case *int8:
			val, err := strconv.ParseInt(field.defaultValue, 10, 8)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a int8 val=%v defaultValue=%v", field.name, val, field.defaultValue)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Int8(lFirst, *ptr, field.usage)
		case *int32:
			val, err := strconv.ParseInt(field.defaultValue, 10, 32)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a int32 val=%v defaultValue=%v", field.name, val, field.defaultValue)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Int32(lFirst, *ptr, field.usage)
		case *int64:
			val, err := strconv.ParseInt(field.defaultValue, 10, 64)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a int64 val=%v defaultValue=%v", field.name, val, field.defaultValue)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Int64(lFirst, *ptr, field.usage)
		case *uint:
			val, err := strconv.Atoi(field.defaultValue)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a uint", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Uint(lFirst, *ptr, field.usage)
		case *uint8:
			val, err := strconv.ParseUint(field.defaultValue, 10, 8)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a uint8", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Uint8(lFirst, *ptr, field.usage)
		case *uint16:
			val, err := strconv.ParseUint(field.defaultValue, 10, 16)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a uint16", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Uint16(lFirst, *ptr, field.usage)
		case *uint32:
			val, err := strconv.ParseUint(field.defaultValue, 10, 32)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a uint32", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Uint32(lFirst, *ptr, field.usage)
		case *uint64:
			val, err := strconv.ParseUint(field.defaultValue, 10, 64)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a uint64", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Uint64(lFirst, *ptr, field.usage)
		case *float32:
			val, err := strconv.ParseFloat(field.defaultValue, 32)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a float32", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Float32(lFirst, *ptr, field.usage)
		case *float64:
			val, err := strconv.ParseFloat(field.defaultValue, 64)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a float64", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Float64(lFirst, *ptr, field.usage)
		case *bool:
			val, err := strconv.ParseBool(field.defaultValue)
			if err != nil {
				cm.logger.Fatalf("default specified for %s is not a bool", field.name)
			}
			viper.SetDefault(lFirst, val)
			cmd.PersistentFlags().Bool(lFirst, *ptr, field.usage)
		case pflag.Value:
			// Any type implementing pflag.Value will be automatically supported
			viper.SetDefault(lFirst, field.defaultValue)
			cmd.PersistentFlags().Var(ptr, lFirst, field.usage)
		default:
			cm.logger.Warnf("unknown field %s type %v", field.name, reflect.TypeOf(field))
		}

		underscoreField := camelCaseToUnderscore(name)
		switch field.namespace {
		case environmentVariable:
			if field.env != "" {
				viper.BindEnv(lFirst, strings.ToUpper(field.env))
			} else {
				viper.BindEnv(lFirst, underscoreField)
			}
		case configURL:
			if field.url != "" {
				viper.BindEnv(lFirst, field.url)
			} else {
				return fmt.Errorf("url tag can't be empty")
			}
		}
		viper.BindPFlag(lFirst, cmd.PersistentFlags().Lookup(lFirst))
	}
	return nil
}

// CreateCommandLineFlags will create command line flags for given config via Cobra and Viper
// to support command line overriding of config values
func (cm *Manager) CreateCommandLineFlags(config interface{}) error {
	return cm.createCommandLineFlags(cm.rootCmd, config)
}

type field struct {
	addr         interface{}
	name         string
	defaultValue string
	namespace    namespace
	usage        string
	env          string
	url          string
	required     bool
}

// Turn the first character in a camel case string to lowercase
// If there are more than one uppercase characters then convert
// all of them except for the last to lowercase
func lowerFirst(s string) string {
	result := []rune{}
	backtrack := []rune{}
	for i, x := range s {
		if unicode.IsUpper(x) {
			if i == 0 {
				result = append(result, unicode.ToLower(x))
			} else {
				backtrack = append(backtrack, x)
			}
		} else {
			// go back and take all out of backtrack and convert it to lower
			// and append to result, except for the last one
			if len(backtrack) > 0 {
				for j := 0; j < len(backtrack)-1; j++ {
					result = append(result, unicode.ToLower(backtrack[j]))
				}
				result = append(result, backtrack[len(backtrack)-1])
				backtrack = []rune{}
			}
			result = append(result, x)
		}
	}
	if len(backtrack) > 0 {
		for _, v := range backtrack {
			result = append(result, unicode.ToLower(v))
		}
	}
	return string(result)
}

func getAllFields(obj interface{}, m map[string]field) {
	xv := reflect.ValueOf(obj).Elem() // Dereference into addressable value
	xt := xv.Type()

	for i := 0; i < xt.NumField(); i++ {
		f := xt.Field(i)
		if f.Anonymous {
			getAllFields(xv.Field(i).Addr().Interface(), m)
		} else {
			addr := xv.Field(i).Addr().Interface()

			fld := getField(xt.Field(i))
			fld.name = f.Name
			fld.addr = addr
			// add defaults to help, cobra/viper doesn't let us add this
			// and no clear example on how to use SetHelpTemplate
			fld.usage = fmt.Sprintf("%s [default: %v]", fld.usage, fld.defaultValue)
			m[f.Name] = fld
		}
	}
}

func getField(fld reflect.StructField) field {
	f := field{
		required: false,
	}
	tags := strings.Split(fld.Tag.Get(tagName), ";")
	for _, tag := range tags {
		t := strings.Split(tag, "=")
		if t[0] == "default" {
			f.defaultValue = t[1]
		} else if t[0] == "usage" {
			f.usage = t[1]
		} else if t[0] == string(environmentVariable) {
			f.env = t[1]
			f.namespace = environmentVariable
		} else if t[0] == "required" {
			f.required = true
		} else if t[0] == "url" {
			f.url = t[1]
			f.namespace = configURL
		}

	}
	return f
}

func camelCaseToUnderscore(s string) string {
	var out []string
	for _, parts := range camelCaseRegex.FindAllStringSubmatch(s, -1) {
		if parts[1] != "" {
			out = append(out, parts[1])
		}
		if parts[2] != "" {
			out = append(out, parts[2])
		}
	}
	return strings.ToUpper(strings.Join(out, "_"))
}

// StdinConfigLoader provides autowiring of config values piped in from stdin
type StdinConfigLoader struct{}

// Load trigger recursive load of config values from yaml piped to stdin
func (s *StdinConfigLoader) Load(config interface{}) error {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(os.Stdin)

		viper.SetConfigType("yaml")
		if err := viper.ReadConfig(buf); err != nil {
			return err
		}
	}
	return nil
}
