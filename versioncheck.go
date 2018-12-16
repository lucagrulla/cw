package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/fatih/color"
)

func fetchLatestVersion() chan string {
	latestVersionChannel := make(chan string, 1)
	go func() {
		r, e := http.Get("https://github.com/lucagrulla/cw/releases/latest")

		if e != nil {
			close(latestVersionChannel)
		} else {
			finalURL := r.Request.URL.String()
			tokens := strings.Split(finalURL, "/")
			latestVersionChannel <- tokens[len(tokens)-1]
		}
	}()
	return latestVersionChannel
}

func newVersionMsg(currentVersion string, latestVersionChannel chan string, noColor bool) {
	latestVersion, ok := <-latestVersionChannel
	//if the channel is closed it means we failed to fetch the latest version. Ignore the version message.
	if !ok {
		if latestVersion != fmt.Sprintf("v%s", currentVersion) {
			fmt.Println("")
			fmt.Println("")
			if noColor {
				msg := fmt.Sprintf("%s - %s -> %s", "A new version of cw is available!", currentVersion, latestVersion)
				fmt.Println(msg)
			} else {
				msg := fmt.Sprintf("%s - %s -> %s", color.GreenString("A new version of cw is available!"), color.YellowString(currentVersion), color.GreenString(latestVersion))
				fmt.Println(msg)
			}
		}
	}
}

func versionCheckOnSigterm() {
	//only way to avoid print of the signal: interrupt message
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	os.Exit(0)
}
