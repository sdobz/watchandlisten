package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"os/exec"
	"bytes"
	"os"
	"io"
	"errors"
)

type WALConfigHook struct {
	Webhook_url string
	Events      []string
	Command     string
	Ref         string
}

func (hook *WALConfigHook) appliesTo(r *http.Request) bool {
	events, ok := r.Header["X-Github-Event"];
	if  !ok || len(events) != 1 {
		log.Print("No X-Github-Event header found")
		return false
	}

	event := events[0]
	found := false
	for _, e := range hook.Events {
		found = found || event == e
	}
	if !found {
		log.Printf("%s not interested in %s", r.URL.Path, event)
		return false
	}

	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)

	var payload WebhookPayload
	err := decoder.Decode(&payload)
	if err != nil {
		log.Print(err)
		return false
	}

	if payload.Ref != hook.Ref {
		log.Printf("Ignoring ref: %s", payload.Ref)
		return false
	}

	return true
}

func (hook *WALConfigHook) run() error {
	cmd := exec.Command("bash", "-c", hook.Command)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	log.Print("> ", hook.Command)
	if err != nil {
		log.Print(err)
	}
	log.Print(out.String())
	return nil
}

type WALConfig struct {
	Log   string
	Addr  string
	RunWebhook string `json:""`
	Hooks []WALConfigHook
}

func (config *WALConfig) findHook(path string) (*WALConfigHook, error) {
	for _, hook := range config.Hooks {
		if hook.Webhook_url != path {
			continue
		}
		return &hook, nil
	}
	return nil, errors.New("Hook not found")
}

type WebhookPayload struct {
	Ref string
}

type EventResponse struct {
	Events []Event
}

type Event struct {
	Type string
	Payload WebhookPayload
}

func getHandler(config *WALConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		log.Printf("%s %s", r.Method, path)

		if path == "/" {
			logf, err := os.OpenFile(config.Log, os.O_RDONLY, 0640)
			if err != nil {
				log.Print(err)
				return
			}
			defer logf.Close()

			_, err = io.Copy(w, logf)
			if err != nil {
				log.Print(err)
				return
			}
		} else if r.Method == "POST" {
			hook, err := config.findHook(path)

			if err != nil {
				log.Print(err.Error())
				errorHandler(w, r, http.StatusNotFound)
				return
			}

			if !hook.appliesTo(r) {
				return
			}

			if err = hook.run(); err != nil {
				log.Print(err)
			}
		} else {
			errorHandler(w, r, http.StatusNotFound)
		}
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "404")
	}
}

func getConfig() (*WALConfig, error) {
	testMode := flag.Bool("test", false, "Whether to test configuration")
	configPath := flag.String("conf", "/etc/watchandlisten/conf.json", "Configuration location")
	runWebhook := flag.String("run", "", "Run specified webhook and exit")
	flag.Parse()

	raw, err := ioutil.ReadFile(*configPath)
	if err != nil {
		if *testMode {
			fmt.Print("Error loading config:\n")
			fmt.Print(err, "\n")
			os.Exit(1)
		}
		return nil, err
	}

	var config WALConfig
	err = json.Unmarshal(raw, &config)
	if err != nil {
		if *testMode {
			fmt.Print("Error parsing config:\n")
			fmt.Print(err, "\n")
			os.Exit(1)
		}
		return nil, err
	}

	if *testMode {
		cmd := exec.Command("test", "-w", config.Log)

		err = cmd.Run()
		if err != nil {
			fmt.Print("Cannot write log file")
			os.Exit(1)
		}
		fmt.Print("OK\n")
		os.Exit(0)
	}

	config.RunWebhook = *runWebhook

	return &config, nil
}

func main() {
	conf, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	if conf.RunWebhook != "" {
		hook, err := conf.findHook(conf.RunWebhook)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		if err = hook.run(); err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	logf, err := os.OpenFile(conf.Log, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Fatal(err)
	}
	defer logf.Close()
	log.SetOutput(io.MultiWriter(logf, os.Stderr))

	log.Printf("Serving on: %s", conf.Addr)
	log.Printf("Log location: %s", conf.Log)
	log.Print("Registered webhooks:")
	for _, hook := range conf.Hooks {
		log.Printf("  %s %s:", hook.Webhook_url, hook.Events)
		log.Printf("    %s", hook.Ref)
		log.Printf("    %s", hook.Command)
	}

	http.HandleFunc("/", getHandler(conf))
	http.ListenAndServe(conf.Addr, nil)
}
