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
	"unsafe"
)

/*
编译为linux可用的程序
set GOARCH=amd64
set GOOS=linux
go build
*/

func main() {
	//addTest()
	delTest()
}

func addTest() {
	Test("preLogin")
}

func delTest() {
	Test("logout")
}

func Test(kind string) {
	var (
		count = 0
		ip    string
		group = "60012310"
		str   *string
	)
	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			//result := httpPost(url,
			//	"55.55."+strconv.Itoa(i)+"."+strconv.Itoa(j),
			//	"weixin")
			ip = "55.55." + strconv.Itoa(i) + "." + strconv.Itoa(j)
			resp, err := http.Get(fmt.Sprintf("http://10.154.55.20:9800/change-group?kind=%s&userIp=%s&userGroupName=%s", kind, ip, group))
			if err != nil {
				fmt.Printf("count%d，result---------------->%s\n", count, err.Error())
				count++
				continue
			}
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("count%d，status not ok ---------------->status%d\n", count, resp.StatusCode)
				count++
				continue
			}
			bytes, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
			str = (*string)(unsafe.Pointer(&bytes))
			if err != nil {
				fmt.Printf("count%d，result---------------->%s\n", count, err.Error())
			} else {
				if *str == "1:msg" {
					fmt.Printf("count%d，result：%s\n", count, *str)
				} else {
					fmt.Printf("count%d，msg---------------->%s\n", count, *str)
				}
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
