package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go_firewall/cmder"
	"io/ioutil"
	"net/http"
)

type tokenResult struct {
	Access_token string
	Expires_in   int
}

type ipList struct {
	Ip_list []string
}

func main() {
	const urlGetToken = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=wx34553cb2fd56ddb3&secret=ed4b50a231710fe19eafd2a545f9df87"
	data, err := tokenFetch(urlGetToken)
	if err != nil {
		fmt.Println(err)
	}

	UrlIpList := "https://api.weixin.qq.com/cgi-bin/getcallbackip?access_token=" + data.Access_token
	iplist, err := weixinIpFetch(UrlIpList)
	if err != nil {
		panic(err)
	}
	fmt.Println(iplist.Ip_list)
	var cmd string
	//遍历在本机执行shell，将ip添加到ipset的Weixin组中
	for _, ip := range iplist.Ip_list {
		cmd = "ipset add Weixin " + ip
		_, err := cmder.Exec_shell(cmd)
		if err != nil {
			fmt.Println(err)
		}
	}

}

//获取token，返回Unmarshal后的tokenResult类
func tokenFetch(url string) (tokenResult, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	request.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36",
	)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return tokenResult{}, fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	bodyReader := bufio.NewReader(resp.Body)
	data, err := ioutil.ReadAll(bodyReader)

	token := tokenResult{}
	jerr := json.Unmarshal(data, &token)
	if jerr != nil {
		panic(jerr)
	}
	return token, nil

}

//获取微信服务器IP列表，返回Unmarshal后的ipList类
func weixinIpFetch(url string) (ipList, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	request.Header.Add(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36",
	)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ipList{}, fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	bodyReader := bufio.NewReader(resp.Body)
	data, err := ioutil.ReadAll(bodyReader)
	iplist := ipList{}
	json.Unmarshal(data, &iplist)
	return iplist, nil

}
