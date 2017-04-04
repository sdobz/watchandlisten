package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"errors"
	"os/exec"
	"bytes"
	"os"
	"io"
)

type WALConfigHook struct {
	Webhook_url string
	Events      []string
	Command     string
	Ref         string
}

type WALConfig struct {
	Log   string
	Addr  string
	Hooks []WALConfigHook
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
		log.Printf("%s", path)

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
			var thisHook WALConfigHook

			for _, hook := range config.Hooks {
				if hook.Webhook_url != path {
					continue
				}
				thisHook = hook
			}

			if &thisHook == nil {
				log.Printf("POST to unknown hook: %s", path)
				errorHandler(w, r, http.StatusNotFound)
				return
			}

			events, ok := r.Header["X-Github-Event"];
			if  !ok || len(events) != 1 {
				log.Print("No X-Github-Event header found")
				return
			}

			event := events[0]
			found := false
			for _, e := range thisHook.Events {
				found = found || event == e
			}
			if !found {
				log.Printf("%s not interested in %s", path, event)
				return
			}

			defer r.Body.Close()
			decoder := json.NewDecoder(r.Body)

			var payload WebhookPayload
			err := decoder.Decode(&payload)
			if err != nil {
				log.Print(err)
				return
			}

			if payload.Ref != thisHook.Ref {
				log.Printf("Ignoring ref: %s", payload.Ref)
			}

			cmd := exec.Command("bash", "-c", thisHook.Command)

			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			err = cmd.Run()
			log.Print("> ", thisHook.Command)
			if err != nil {
				log.Print(err)
			}
			log.Print(out.String())
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
	flag.Parse()
	if flag.NArg() != 1 {
		return nil, errors.New("Usage: watchandlisten <conf.json>")
	}

	configPath := flag.Arg(0)

	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config WALConfig
	err = json.Unmarshal(raw, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	conf, err := getConfig()
	if err != nil {
		log.Fatal(err)
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
