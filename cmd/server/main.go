package main

import (
	"context"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"tgracchus/numbers"
)

func main() {
	pflag.Bool("profile", false, "profile the server")
	pflag.Int("concurrent-connections", 5, "number of concurrent connections")
	pflag.String("port", "4000", "tcp port where to start the server")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatal(err)
	}
	pflag.Parse()
	connections := viper.GetInt("concurrent-connections")
	port := viper.GetString("port")
	profile := viper.GetBool("profile")
	log.Printf("profile: %t", profile)
	if profile {
		f, err := os.Create("numbers_cpu.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	numbers.StartNumberServer(context.Background(), connections, "localhost:"+port)
}
