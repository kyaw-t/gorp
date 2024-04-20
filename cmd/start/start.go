package start

import (
	"fmt"
	"gorp/internals/config"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

type InitMessageOptions struct {
	configFile   string
	dryRun       bool
	serverConfig config.GorpConfig
}

func printInitMessage(options InitMessageOptions) {
	if options.dryRun {
		fmt.Printf("Warning: Dry run enabled\n")
		fmt.Printf("All network requests to download artifacts will be mocked and return 200\n")
	}
	fmt.Printf("Running Gorp Server\n Listening on %s:%d\n", "localhost", options.serverConfig.Server.Port)
}

func initServerConfig(configFilePtr *string, portPtr *int, dryRunPtr *bool) (config.GorpConfig, error) {
	configFile, err := config.GetConfigFile(*configFilePtr)
	if err != nil {
		fmt.Printf("error getting config file: %s\n", err)
		return config.GorpConfig{}, err
	}

	serverConfig, err := config.LoadConfigFile(configFile, config.Overrides{Port: *portPtr})
	if err != nil {
		fmt.Printf("error loading config file: %s\n%s", configFile, err)
		return serverConfig, err
	}

	initMessageOptions := InitMessageOptions{
		configFile:   configFile,
		serverConfig: serverConfig,
		dryRun:       *dryRunPtr,
	}
	defer printInitMessage(initMessageOptions)

	return serverConfig, nil

}

func Start(configFilePtr *string, portPtr *int, dryRunPtr *bool) {

	serverConfig, err := initServerConfig(configFilePtr, portPtr, dryRunPtr)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err != nil {
		logger.Error("Error initializing server config\n exiting...\n")
		return
	}
	port := strconv.Itoa(serverConfig.Server.Port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Hello from Gorp Server")
	})

	// Node proxy endpoint
	nodeHandler := &NodeHandler{nodeConfig: serverConfig.Node, logger: *logger}
	http.HandleFunc("/node/", nodeHandler.ServeHTTP)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Gorp Server is healthy\n")
	})

	http.ListenAndServe(":"+port, nil)
}
