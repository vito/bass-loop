package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	flag "github.com/spf13/pflag"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/zapctx"
)

var flags = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

var httpAddr string
var sshAddr string

var githubWebURL string
var githubV3APIURL string
var githubAppID int64
var githubAppPrivateKey string
var githubAppWebhookSecret string

var profPort int
var profFilePath string

var showHelp bool
var showVersion bool

func init() {
	flags.SetOutput(os.Stdout)
	flags.SortFlags = false

	flags.StringVar(&httpAddr, "http", "0.0.0.0:8080", "address on which to listen for HTTP traffic")
	flags.StringVar(&sshAddr, "ssh", "0.0.0.0:6455", "address on which to listen for SSH traffic")

	flags.StringVar(&githubWebURL, "github-url", "https://github.com", "GitHub web URL")
	flags.StringVar(&githubV3APIURL, "github-v3-api-url", "https://api.github.com", "GitHub v3 API URL")

	flags.Int64Var(&githubAppID, "github-app-id", 0, "GitHub app ID")
	flags.StringVar(&githubAppPrivateKey, "github-app-key", "", "path to GitHub app private key")
	flags.StringVar(&githubAppWebhookSecret, "github-app-webhook-secret", "", "secret to verify for GitHub app webhook payloads")

	flags.IntVar(&profPort, "profile", 0, "port number to bind for Go HTTP profiling")
	flags.StringVar(&profFilePath, "cpu-profile", "", "take a CPU profile and save it to this path")

	flags.BoolVarP(&showVersion, "version", "v", false, "print the version number and exit")
	flags.BoolVarP(&showHelp, "help", "h", false, "show bass usage and exit")
}

func main() {
	logger := bass.Logger()
	ctx := zapctx.ToContext(context.Background(), logger)
	ctx = bass.WithTrace(ctx, &bass.Trace{})
	ctx = ioctx.StderrToContext(ctx, os.Stderr)

	err := flags.Parse(os.Args[1:])
	if err != nil {
		cli.WriteError(ctx, bass.FlagError{
			Err:   err,
			Flags: flags,
		})
		os.Exit(2)
		return
	}

	err = root(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func root(ctx context.Context) error {
	if showVersion {
		printVersion(ctx)
		return nil
	}

	if showHelp {
		help(ctx)
		return nil
	}

	if profPort != 0 {
		zapctx.FromContext(ctx).Sugar().Debugf("serving pprof on :%d", profPort)

		l, err := net.Listen("tcp", fmt.Sprintf(":%d", profPort))
		if err != nil {
			cli.WriteError(ctx, err)
			return err
		}

		go http.Serve(l, nil)
	}

	if profFilePath != "" {
		profFile, err := os.Create(profFilePath)
		if err != nil {
			cli.WriteError(ctx, err)
			return err
		}

		defer profFile.Close()

		pprof.StartCPUProfile(profFile)
		defer pprof.StopCPUProfile()
	}

	return httpServe(ctx)
}
