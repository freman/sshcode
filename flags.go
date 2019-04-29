package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/Bob-Thomas/configdir"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func flags() (host string) {
	cfgFile := pflag.StringP("config", "C", "", "Configuration file for sshcode")
	pflag.StringP("identity", "i", "", "Identity file (eg: ~/.ssh/id_rsa")
	pflag.StringP("login", "l", "", "Login username")
	pflag.StringP("bind", "b", "", "Bind address")
	pflag.IntP("port", "p", 22, "Port")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-b bind_address] [-i identity_file] [user@]host[:port] [-l login_name] [-p port]\n", path.Base(os.Args[0]))
		pflag.PrintDefaults()
	}

	pflag.Parse()
	for _, flagName := range []string{"identity", "login", "bind", "port"} {
		viper.BindPFlag(flagName, pflag.Lookup(flagName))
	}

	var loginPassed, portPassed, configPassed bool

	pflag.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "login":
			loginPassed = true
		case "port":
			portPassed = true
		case "config":
			configPassed = true
		}
	})

	viper.SetDefault("workdir", "~")

	viper.SetEnvPrefix("sshcode")
	viper.AutomaticEnv()

	if configPassed && *cfgFile != "" {
		viper.SetConfigFile(*cfgFile)
	} else if tmp := os.Getenv("SSHCODE_CONFIG"); tmp != "" {
		viper.SetConfigFile(tmp)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(configdir.SystemSettingsDir("freman", "sshcode"))
		viper.AddConfigPath(configdir.SettingsDir("freman", "sshcode"))
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, isa := err.(viper.ConfigFileNotFoundError); !isa {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	arg := pflag.Arg(0)
	if arg == "" {
		pflag.Usage()
		os.Exit(1)
	}

	if i := strings.Index(arg, "@"); i >= 0 {
		if !loginPassed {
			viper.Set("login", arg[:i])
		}
		arg = arg[i+1:]
	}

	var (
		port string
		err  error
	)

	host, port, err = net.SplitHostPort(arg)
	if err != nil {
		if !strings.HasSuffix(err.Error(), "missing port in address") {
			panic(err)
		}
	}

	if !portPassed && port != "" {
		p, _ := strconv.Atoi(port)
		viper.Set("port", p)
	}

	workDir := pflag.Arg(1)
	if workDir != "" {
		viper.Set("workdir", workDir)
	}

	if host == "" {
		return arg
	}

	return host
}
