package main

import (
	"flag"
	"gopkg.in/tylerb/graceful.v1"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
	"strconv"
)

var (
	statusLineRegexp = regexp.MustCompile(`(?m)^(.*):\s+(.*)$`)
	fpmStatusURL     = ""
)

func main() {
	url := flag.String("status-url", "", "PHP-FPM status URL")
	addr := flag.String("addr", "0.0.0.0:8080", "IP/port for the HTTP server")
	flag.Parse()

	if *url == "" {
		log.Fatal("The status-url flag is required.")
	} else {
		fpmStatusURL = *url
	}

	scrapeFailures := 0

	server := &graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:        *addr,
			ReadTimeout: time.Duration(5) * time.Second,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp, err := http.Get(fpmStatusURL)

				if err != nil {
					log.Println(err)
					scrapeFailures = scrapeFailures+1
					x := strconv.Itoa(scrapeFailures)
					NewMetricsFromMatches([][]string{{"scrape failure:","scrape failure",x}}).WriteTo(w)
					return
				}

				if (resp.StatusCode != http.StatusOK){
					log.Println("php-fpm status code is not OK.")
					scrapeFailures = scrapeFailures+1
					x := strconv.Itoa(scrapeFailures)
					NewMetricsFromMatches([][]string{{"scrape failure:","scrape failure",x}}).WriteTo(w)
					return
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Println(err)
					scrapeFailures = scrapeFailures+1
					x := strconv.Itoa(scrapeFailures)
					NewMetricsFromMatches([][]string{{"scrape failure:","scrape failure",x}}).WriteTo(w)
					return
				}

				resp.Body.Close()

				x := strconv.Itoa(scrapeFailures)

				matches := statusLineRegexp.FindAllStringSubmatch(string(body), -1)
				matches = append(matches,[]string{"scrape failure:","scrape failure",x})

				NewMetricsFromMatches(matches).WriteTo(w)
			}),
		},
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
