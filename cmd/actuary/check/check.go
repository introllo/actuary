package check

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"github.com/diogomonica/actuary/actuary"
	"github.com/diogomonica/actuary/oututils"
	"github.com/diogomonica/actuary/profileutils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var profile string
var output string
var tlsPath string
var server string
var dockerServer string
var tomlProfile profileutils.Profile
var results []actuary.Result
var actions map[string]actuary.Check

type Request struct {
	NodeID  []byte
	Results []byte
}

func init() {
	CheckCmd.Flags().StringVarP(&profile, "profile", "f", "", "file profile")
	CheckCmd.Flags().StringVarP(&output, "output", "o", "", "output filename")
	CheckCmd.Flags().StringVarP(&tlsPath, "tlsPath", "t", "", "Path to load certificates from")
	CheckCmd.Flags().StringVarP(&server, "server", "s", "", "Server for aggregating results")
	CheckCmd.Flags().StringVarP(&dockerServer, "dockerServer", "d", "", "Docker server to connect to tcp://<docker host>:<port>")
}

func HttpClient() (client *http.Client) {
	uckey := os.Getenv("X509_USER_KEY")
	ucert := os.Getenv("X509_USER_CERT")
	x509cert, err := tls.LoadX509KeyPair(ucert, uckey)
	if err != nil {
		panic(err.Error())
	}
	certs := []tls.Certificate{x509cert}
	if len(certs) == 0 {
		client = &http.Client{}
		return
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: certs,
			InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
	return
}

// Log in and retrieve token
func basicAuth(client *http.Client) string {
	req, err := http.NewRequest("GET", "https://server:8000/token", nil)
	if err != nil {
		log.Fatalf("Error generating request: %v", err)
	}
	var pw []byte
	pw, err = ioutil.ReadFile(os.Getenv("TOKEN_PASSWORD"))
	if err != nil {
		log.Fatalf("Could not read password: %v", err)
	}
	req.SetBasicAuth("defaultUser", string(pw))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Basic auth: %v", err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Status code: %v", resp.StatusCode)
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	s := string(bodyText)
	return s
}

var (
	CheckCmd = &cobra.Command{
		Use:   "check <server name>",
		Short: "Run actuary checklist on a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			urlPOST := server
			var cmdArgs []string
			var hash string
			if tlsPath != "" {
				os.Setenv("DOCKER_CERT_PATH", tlsPath)
			}
			if dockerServer != "" {
				os.Setenv("DOCKER_HOST", dockerServer)
			} else {
				os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
			}
			trgt, err := actuary.NewTarget()
			if err != nil {
				log.Fatalf("Unable to connect to Docker daemon: %s", err)
			}
			cmdArgs = flag.Args()
			if len(cmdArgs) == 2 {
				hash = cmdArgs[1]
				tomlProfile, err = profileutils.GetFromURL(hash)
				if err != nil {
					log.Fatalf("Unable to fetch profile. Exiting...")
				}
			} else if len(cmdArgs) == 0 || len(cmdArgs) == 1 {
				_, err := os.Stat(profile)
				if os.IsNotExist(err) {
					log.Fatalf("Invalid profile path: %s", profile)
				}
				tomlProfile = profileutils.GetFromFile(profile)
			} else {
				log.Fatalf("Unsupported number of arguments. Use -h for help")
			}
			actions := actuary.GetAuditDefinitions()
			for category := range tomlProfile.Audit {
				checks := tomlProfile.Audit[category].Checklist
				for _, check := range checks {
					if _, ok := actions[check]; ok {
						res := actions[check](trgt)
						results = append(results, res)
					} else {
						log.Panicf("No check named %s", check)
					}
				}
			}
			rep := oututils.CreateReport(output)
			rep.Results = results
			switch strings.ToLower(output) {
			case "json":
				rep.WriteJSON()
			case "xml":
				rep.WriteXML()
			default:
				for _, res := range rep.Results {
					oututils.ConsolePrint(res)
				}
			}
			if err != nil {
				log.Fatalf("Unable to marshal node ID")
			}
			jsonResults, err := json.MarshalIndent(rep.Results, "", "  ")
			if err != nil {
				log.Fatalf("Unable to marshal results into JSON file")
			}
			var reqStruct = Request{NodeID: []byte(os.Getenv("NODE")), Results: jsonResults}
			result, err := json.Marshal(reqStruct)

			if err != nil {
				log.Fatalf("Could not marshal request: %v", err)
			}
			reqPost, err := http.NewRequest("POST", urlPOST, bytes.NewBuffer(result))
			if err != nil {
				log.Fatalf("Could not create a new request: %v", err)
			}
			if err != nil {
				log.Fatalf("couldn't read req: %v", err)
			}
			reqPost.Header.Set("Content-Type", "application/json")
			client := HttpClient()

			token := basicAuth(client)
			var bearer = "Bearer " + token
			reqPost.Header.Add("authorization", bearer)
			respPost, err := client.Do(reqPost)

			if err != nil {
				log.Fatalf("Could not send post request to client: %v", err)
			}
			defer respPost.Body.Close()
			return nil
		},
	}
)
