package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

//Config struct holds all configuration data that comes from config.yml or environment variables
type Config struct {
	SecurityTrails struct {
		Key string `yaml:"key" envconfig:"SECURITYTRAILS_KEY"`
	} `yaml:"securitytrails"`
}

var config *Config
var apiEndpoint string
var output string

func main() {
	// default config file location
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error finding home dir:", err)
		return
	}
	defaultConfigFile := home + "/.config/haktools/haktrails-config.yml"

	if len(os.Args) <= 1 {
		help()
		os.Exit(1)
	}

	// parse the command line options
	mainFlagSet := flag.NewFlagSet("haktrails", flag.ExitOnError)
	concurrencyPtr := mainFlagSet.Int("t", 2, "Number of threads to utilise. Keep in mind that the API has rate limits.")
	buffer := mainFlagSet.Int("b", 100, "Number of subdomains to send per request for the submission endpoint")
	configFile := mainFlagSet.String("c", defaultConfigFile, "Config file location")
	outputPointer := mainFlagSet.String("o", "list", "output format, list or json. json will return the raw data from SecurityTrails")
	lookupType := mainFlagSet.String("type", "a", "DNS record type (only used for historical DNS queries): a,aaaa,mx,ns,soa,txt")
	query := mainFlagSet.String("query", "", "Query for DSL (https://docs.securitytrails.com/docs/how-to-use-the-dsl)")

	mainFlagSet.Parse(os.Args[2:])

	output = *outputPointer
	// color start
	color.Set(color.FgRed)
	// load config file
	f, err := os.Open(*configFile)
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return
	}
	defer f.Close()
	// parse the config file
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error decoding config file:", err)
		return
	}
	//color stop
	color.Unset()

	apiEndpoint = "https://api.securitytrails.com/v1/" // global

	numWorkers := *concurrencyPtr
	work := make(chan string)
	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			work <- s.Text()
		}
		close(work)
	}()

	wg := &sync.WaitGroup{}

	switch os.Args[1] {
	case "banner":
		banner()
	case "submit":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go submit(work, wg, *buffer)
		}
		wg.Wait()
	case "dsl":
		dsl(*query)
	case "historicalwhois":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go historicalwhois(work, wg)
		}
		wg.Wait()
	case "historicaldns":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go historicaldns(work, wg, *lookupType)
		}
		wg.Wait()
	case "tags":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go tags(work, wg)
		}
		wg.Wait()
	case "whois":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go whois(work, wg)
		}
		wg.Wait()
	case "company":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go company(work, wg)
		}
		wg.Wait()
	case "details":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go details(work, wg)
		}
		wg.Wait()
	case "ping":
		ping()
	case "usage":
		usage()
	case "subdomains":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go subdomains(work, wg)
		}
		wg.Wait()
	case "associateddomains":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go associatedDomains(work, wg)
		}
		wg.Wait()
	case "associatedips":
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go associatedIPs(work, wg)
		}
		wg.Wait()
	// no valid subcommand found - default to showing a message and exiting
	default:
		help()
		os.Exit(1)
	}
}

func help() {

	//color start
	color.Set(color.FgYellow)

	fmt.Println(`Usage incorrect. Hint:

	Subdomains:		cat domains.txt | haktrails subdomains
	Associated domains:	cat domains.txt | haktrails associateddomains
	Associated ips: 	cat domains.txt | haktrails associatedips
	Associated company: 	cat domains.txt | haktrails company
	Historical DNS data:	cat domains.txt | haktrails historicaldns -type <lookuptype>
	Historical whois data:	cat domains.txt | haktrails historicalwhois
	Domain details: 	cat domains.txt | haktrails details
	Domain tags: 		cat domains.txt | haktrails tags
	Whois data: 		cat domains.txt | haktrails whois
        DSL query:              haktrails dsl -query "query here"
	SecurityTrails usage: 	haktrails usage
	Check API Key: 		haktrails ping
	Show the banner:	haktrails banner

	Full details at: https://github.com/trhacknon/haktrails
	`)

	//color stop
	color.Unset()
}

func banner() {
	// color start
	color.Set(color.FgGreen)
	fmt.Println(`
	 _       _   _           _ _     
	| |_ ___| |_| |_ ___ ___|_| |___ 
	|   | .'| '_|  _|  _| .'| | |_ -|
	|_|_|__,|_,_|_| |_| |__,|_|_|___|
									 
	   Made with <3 by trhacknon
	    Sponsored by Anonymous
	         hakluke.com
                                                          
	`)
	//color stop
	color.Unset()
}
