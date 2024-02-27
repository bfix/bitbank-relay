//----------------------------------------------------------------------
// This file is part of 'bitbank-relay'.
// Copyright (C) 2021-2024, Bernd Fix  >Y<
//
// 'bitbank-relay' is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// 'bitbank-relay' is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"relay/lib"
	"syscall"
	"text/template"
	"time"

	"github.com/bfix/gospel/logger"
)

//go:embed gui.htpl
var fsys embed.FS

var (
	cfg     *lib.Config
	mdl     *lib.Model
	tpl     *template.Template
	verbose bool
	Version string = "v0.0.0"
)

func main() {
	// welcome
	defer logger.Flush()
	logger.Println(logger.INFO, "===============================")
	logger.Println(logger.INFO, "bitbank-relay-play "+Version)
	logger.Println(logger.INFO, "(c) 2021-2024, Bernd Fix    >Y<")
	logger.Println(logger.INFO, "===============================")

	// parse arguments
	var confFile, listen string
	flag.StringVar(&confFile, "c", "config.json", "Configuration file (default: config.json)")
	flag.StringVar(&listen, "l", "localhost:8082", "Listen address (default: localhost:8082)")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	// read configuration
	var err error
	logger.Println(logger.INFO, "Reading configuration...")
	if cfg, err = lib.ReadConfigFile(confFile); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}

	// connect to model
	logger.Println(logger.INFO, "Connecting to model...")
	if mdl, err = lib.Connect(cfg.Model); err != nil {
		logger.Println(logger.ERROR, err.Error())
		return
	}
	defer mdl.Close()

	// setup request router
	logger.Println(logger.INFO, "Setting up web service...")
	mux := http.NewServeMux()
	mux.HandleFunc("/account/", accountHandler)
	mux.HandleFunc("/checkout/", payHandler)
	mux.HandleFunc("/", rootHandler)

	// read and prepare templates
	tpl = template.New("gui")
	tpl.Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"trim": func(a float64, b int) string {
			return fmt.Sprintf("%.[2]*[1]f", a, b)
		},
		"valid": func(a interface{}) bool {
			return a != nil
		},
		"date": func(ts int64) string {
			return time.Unix(ts, 0).Format("02 Jan 06 15:04")
		},
	})
	if _, err := tpl.ParseFS(fsys, "gui.htpl"); err != nil {
		logger.Println(logger.ERROR, "GUI templates: "+err.Error())
		return
	}

	// assemble HTTP server
	logger.Printf(logger.INFO, "Starting web GUI at %s", listen)
	srv := &http.Server{
		Handler:      mux,
		Addr:         listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	// start server
	logger.Println(logger.INFO, "Waiting for client requests...")
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Println(logger.ERROR, err.Error())
		}
	}()

	// Prepare context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle OS signals
	sigCh := make(chan os.Signal, 5)
	signal.Notify(sigCh)
loop:
	for sig := range sigCh {
		switch sig {
		case syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM:
			logger.Printf(logger.INFO, "Terminating service (on signal '%s')\n", sig)
			break loop
		case syscall.SIGHUP:
			logger.Println(logger.INFO, "SIGHUP")
		case syscall.SIGURG:
			// TODO: https://github.com/golang/go/issues/37942
		default:
			logger.Println(logger.INFO, "Unhandled signal: "+sig.String())
		}
	}

	// shutdown web service
	ctxSrv, cancelSrv := context.WithTimeout(ctx, 15*time.Second)
	defer cancelSrv()
	srv.Shutdown(ctxSrv)
}
