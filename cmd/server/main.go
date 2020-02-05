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
	pflag.String("port", "1234", "tcp port where to start the server")
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

	if profile {
		f, err := os.Create("numbers_cpu.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		men, err := os.Create("numbers_men.prof")
		if err != nil {
			log.Fatal(err)
		}
		err = pprof.WriteHeapProfile(men)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}

	numbers.StartNumberServer(context.Background(), connections, "localhost:"+port)
}
