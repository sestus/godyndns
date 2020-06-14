package godyndns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

type domain struct {
	Data string `json:"data"`
	Name string `json:"name"`
	Ttl  int    `json:"ttl"`
	Type string `json:"type"`
}

type domainUpdate struct {
	Data string `json:"data"`
	Ttl int 	`json:"ttl"`
}

type domainUpdates []domainUpdate

const domainsPath = "https://api.godaddy.com/v1/domains"

func GetPublicIp(client *http.Client) (net.IP, error) {
	var ipResolvers = [3]string{"http://ipinfo.io/ip", "https://api.ipify.org?format=text", "https://checkip.amazonaws.com/api"}
	for _, url := range ipResolvers {
		ip, err := getPublicIpFrom(client, url)
		if err == nil {
			log.Printf("My public IP is:%s\n", ip)
			return ip, nil
		}
	}
	return nil, fmt.Errorf("couldn't get my public IP. Tried %v", ipResolvers)

}

func UpdateGoDaddyARecord(client *http.Client, domainName string, publicIp net.IP,  apiKey, secretKey string) error {
	if publicIp == nil {
		log.Println("Given publicIp is nll")
		return errors.New("given publicIp is nll")
	}
	url := fmt.Sprintf("%s/mikemylonakis.com/records/A/%s", domainsPath, domainName)
	record, _:= json.Marshal(domainUpdates{domainUpdate{publicIp.String(), 600}})
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(record))
	addHeaders(req, apiKey, secretKey)
	_, err := doRequest(client, req)
	return err

}

func GetGodaddyARecordIp(client *http.Client, domainName string, apiKey, secretKey string) (net.IP, error) {
	url := fmt.Sprintf("%s/mikemylonakis.com/records/A/%s", domainsPath, domainName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Failed to get the record details for domain %s : %s", domainName, err)
		return nil, err
	}
	addHeaders(req, apiKey, secretKey)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to execute request %s : %s", req.URL, err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		log.Printf("%s to %s returned %s.\n", req.Method, req.URL, resp.Status)
		return nil, errors.New(resp.Status)
	}
	defer resp.Body.Close()
	var record []domain
	err = json.NewDecoder(resp.Body).Decode(&record)
	if err != nil {
		log.Printf("Failed to decode the response body. %s", err)
		return nil, err
	}
	ip := net.ParseIP(record[0].Data)
	if ip == nil {
		return ip, fmt.Errorf("couldn't parse %s to an IP address", record[0].Data)
	}
	return net.ParseIP(record[0].Data), nil
}

func addHeaders(r *http.Request, apiKey, secretKey string) *http.Request {
	r.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", apiKey, secretKey))
	r.Header.Set("accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Encoding", "application/json")
	return r
}

func doRequest(client *http.Client, r *http.Request) (string, error) {
	resp, err := client.Do(r)
	if err != nil {
		log.Fatalf("Failed to execute request %s : %s", r.URL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to parse request response : %s", err)
		return "", err
	}
	if resp.StatusCode != 200 {
		log.Printf("%s to %s returned %s.", r.Method, r.URL, resp.Status)
		return "", errors.New(resp.Status)
	}
	return string(body), nil
}

func getPublicIpFrom(client *http.Client, url string) (net.IP, error) {
	log.Printf("Getting my public IP address from  %s ...\n", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("Failed to reach %s to get my public IP address: %s", url, err)
	}
	ret, err := doRequest(client, req)
	if err != nil {
		return nil, err
	} else {
		ip := net.ParseIP(strings.TrimSuffix(ret, "\n"))
		if ip == nil {
			return nil, fmt.Errorf("couldn't parse %s to an IP address", strings.TrimSuffix(ret, "\n"))
		} else {
			return ip, nil
		}
	}
}