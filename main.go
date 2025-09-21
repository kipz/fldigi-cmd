package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

//go:embed bands.txt
var bandPlanData string

type BandRange struct {
	Name     string
	StartMHz float64
	EndMHz   float64
}

var bandPlan []BandRange

func init() {
	loadBandPlan()
}

func loadBandPlan() {
	scanner := bufio.NewScanner(strings.NewReader(bandPlanData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse band:start:end format
		parts := strings.Split(line, ":")
		if len(parts) != 3 {
			continue
		}

		startMHz, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		endMHz, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue
		}

		bandPlan = append(bandPlan, BandRange{
			Name:     parts[0],
			StartMHz: startMHz,
			EndMHz:   endMHz,
		})
	}
}

type FldigiClient struct {
	url    string
	client *http.Client
}

type MethodCall struct {
	XMLName xml.Name `xml:"methodCall"`
	Method  string   `xml:"methodName"`
	Params  *Params  `xml:"params,omitempty"`
}

type Params struct {
	Params []Param `xml:"param"`
}

type Param struct {
	Value Value `xml:"value"`
}

type Value struct {
	String  string `xml:"string,omitempty"`
	Double  string `xml:"double,omitempty"`
	Int     string `xml:"i4,omitempty"`
	Content string `xml:",chardata"`
}

type MethodResponse struct {
	XMLName xml.Name `xml:"methodResponse"`
	Params  *Params  `xml:"params,omitempty"`
	Fault   *Fault   `xml:"fault,omitempty"`
}

type Fault struct {
	Value struct {
		Struct []Member `xml:"struct>member"`
	} `xml:"value"`
}

type Member struct {
	Name  string `xml:"name"`
	Value Value  `xml:"value"`
}

func NewFldigiClient(host string, port int) *FldigiClient {
	url := fmt.Sprintf("http://%s:%d/RPC2", host, port)

	// Create HTTP client with IPv4-only transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	// Force IPv4 by setting up custom dialer
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		// Force tcp4 instead of tcp to use IPv4 only
		if network == "tcp" {
			network = "tcp4"
		}
		return d.DialContext(ctx, network, addr)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &FldigiClient{
		url:    url,
		client: client,
	}
}

func (fc *FldigiClient) ListMethods() error {
	call := MethodCall{
		Method: "system.listMethods",
	}

	xmlData, err := xml.Marshal(call)
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}

	resp, err := fc.client.Post(fc.url, "text/xml", bytes.NewBuffer(xmlData))
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Available methods:\n%s\n", string(body))
	return nil
}

func (fc *FldigiClient) GetFrequency() (float64, error) {
	call := MethodCall{
		Method: "rig.get_vfo",
	}

	xmlData, err := xml.Marshal(call)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal XML: %v", err)
	}

	resp, err := fc.client.Post(fc.url, "text/xml", bytes.NewBuffer(xmlData))
	if err != nil {
		return 0, fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %v", err)
	}

	var response MethodResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Fault != nil {
		return 0, fmt.Errorf("XML-RPC fault occurred. Response: %s", string(body))
	}

	if response.Params == nil || len(response.Params.Params) == 0 {
		return 0, fmt.Errorf("no frequency data in response")
	}

	freqStr := response.Params.Params[0].Value.String
	if freqStr == "" {
		freqStr = response.Params.Params[0].Value.Double
	}
	if freqStr == "" {
		freqStr = response.Params.Params[0].Value.Int
	}
	if freqStr == "" {
		freqStr = response.Params.Params[0].Value.Content
	}

	if freqStr == "" {
		return 0, fmt.Errorf("empty frequency response: %s", string(body))
	}

	freq, err := strconv.ParseFloat(freqStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse frequency '%s': %v", freqStr, err)
	}
	return freq, nil
}

func frequencyToBand(freq float64) string {
	freqMHz := freq / 1000000

	// Check each band in the loaded band plan
	for _, band := range bandPlan {
		if freqMHz >= band.StartMHz && freqMHz <= band.EndMHz {
			return band.Name
		}
	}

	return "unknown"
}

func runExternalCommand(command string, band string) error {
	cmd := exec.Command(command, band)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	var host, command string
	var port int
	var interval time.Duration

	flag.StringVar(&host, "h", "127.0.0.1", "fldigi host")
	flag.StringVar(&host, "host", "127.0.0.1", "fldigi host")
	flag.IntVar(&port, "p", 7362, "fldigi XML-RPC port")
	flag.IntVar(&port, "port", 7362, "fldigi XML-RPC port")
	flag.DurationVar(&interval, "i", 5*time.Second, "polling interval")
	flag.DurationVar(&interval, "interval", 5*time.Second, "polling interval")
	flag.StringVar(&command, "c", "", "external command to run on band change")
	flag.StringVar(&command, "command", "", "external command to run on band change")

	flag.Parse()

	if command == "" {
		fmt.Fprintf(os.Stderr, "Error: --command/-c flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	client := NewFldigiClient(host, port)

	var currentBand string
	fmt.Printf("Starting fldigi band monitor (interval: %v)\n", interval)

	for {
		freq, err := client.GetFrequency()
		if err != nil {
			log.Printf("Error getting frequency: %v", err)
			time.Sleep(interval)
			continue
		}

		band := frequencyToBand(freq)
		if band == "unknown" {
			time.Sleep(interval)
			continue
		}

		if band != currentBand && currentBand != "" {
			fmt.Printf("Band changed from %s to %s (%.3f MHz)\n", currentBand, band, freq/1000000)
			if err := runExternalCommand(command, band); err != nil {
				log.Printf("Error running external command: %v", err)
			}
		} else if currentBand == "" {
			fmt.Printf("Initial band detected: %s (%.3f MHz)\n", band, freq/1000000)
		}

		currentBand = band
		time.Sleep(interval)
	}
}