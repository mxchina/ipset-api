package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func main() {
	//addTest()
	delTest()
}

func addTest() {
	Test("http://172.16.10.80:9800/add")
}

func delTest() {
	Test("http://172.16.10.80:9800/del")
}

func Test(url string) {
	var count = 0
	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			result := httpPost(url,
				"55.55."+strconv.Itoa(i)+"."+strconv.Itoa(j),
				"weixin")
			if result != 0 {
				fmt.Printf("count%d，result---------------->%d\n", count, result)
			} else {
				fmt.Printf("count%d，result：%d\n", count, result)
			}
			count++
		}
	}
}

type PostResult struct {
	Code  int
	Err   string
	Group string
	Ip    string
}

func httpPost(url, ip, group string) int {
	resp, err := http.Post(url,
		"application/x-www-form-urlencoded",
		strings.NewReader("ip="+ip+"&group="+group))
	//strings.NewReader(fmt.Sprintf("ip=%s&group=%s", ip, group)))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("StatusCode" + strconv.Itoa(resp.StatusCode))
		return 1
	}

	body, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
	if err != nil {
		log.Println("ReadAll：" + err.Error())
		return 1
	}

	var postResult PostResult
	err = json.Unmarshal(body, &postResult)
	if err != nil {
		log.Println("Unmarshal：" + err.Error())
		return 1
	}
	return postResult.Code
}
