package main

import (
	"bytes"
	"context"
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
	"time"
)

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

	// Map frequency to actual amateur radio bands
	switch {
	case freqMHz >= 0.1357 && freqMHz <= 0.1378: // 135.7-137.8 kHz
		return "2200m"
	case freqMHz >= 0.472 && freqMHz <= 0.479: // 472-479 kHz
		return "630m"
	case freqMHz >= 1.8 && freqMHz <= 2.0:
		return "160m"
	case freqMHz >= 3.5 && freqMHz <= 4.0:
		return "80m"
	case freqMHz >= 5.3305 && freqMHz <= 5.4035:
		return "60m"
	case freqMHz >= 7.0 && freqMHz <= 7.3:
		return "40m"
	case freqMHz >= 10.1 && freqMHz <= 10.15:
		return "30m"
	case freqMHz >= 14.0 && freqMHz <= 14.35:
		return "20m"
	case freqMHz >= 18.068 && freqMHz <= 18.168:
		return "17m"
	case freqMHz >= 21.0 && freqMHz <= 21.45:
		return "15m"
	case freqMHz >= 24.89 && freqMHz <= 24.99:
		return "12m"
	case freqMHz >= 28.0 && freqMHz <= 29.7:
		return "10m"
	case freqMHz >= 50.0 && freqMHz <= 54.0:
		return "6m"
	case freqMHz >= 144.0 && freqMHz <= 148.0:
		return "2m"
	case freqMHz >= 222.0 && freqMHz <= 225.0:
		return "1.25m"
	case freqMHz >= 420.0 && freqMHz <= 450.0:
		return "70cm"
	case freqMHz >= 902.0 && freqMHz <= 928.0:
		return "33cm"
	case freqMHz >= 1240.0 && freqMHz <= 1300.0:
		return "23cm"
	case freqMHz >= 2300.0 && freqMHz <= 2450.0:
		return "13cm"
	case freqMHz >= 3300.0 && freqMHz <= 3500.0:
		return "9cm"
	case freqMHz >= 5650.0 && freqMHz <= 5925.0:
		return "5cm"
	case freqMHz >= 10000.0 && freqMHz <= 10500.0:
		return "3cm"
	case freqMHz >= 24000.0 && freqMHz <= 24250.0:
		return "1.2cm"
	default:
		return "unknown"
	}
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