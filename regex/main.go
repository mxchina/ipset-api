package main

import (
	"fmt"
	"regexp"
)

func main() {
	const text = `Name123Mem:Nip1\\n\\nNip2NNName345Mem:Nip3Nip4N`
	//weixinRE := regexp.MustCompile(`Name: Weixin.*Members:\\n(.*)\\n`)
	weixinRE := regexp.MustCompile(`(Name123.*Mem:N.*{NName}*.*)`)
	//weixinRE := regexp.MustCompile(`(Name123Mem:Nip1Nip2NNName345Mem:Nip3Nip4N)`)
	weixinRE1 := regexp.MustCompile(`Name123Mem:N(.*)(?:NName.*)*N`)
	weixinRE2 := regexp.MustCompile(`(.*)(NName)?.*N`)

	//const text  = `nameA123nameB456nameC789nameD101112`
	//reg := regexp.MustCompile(`nameB(.*)`)
	result := weixinRE.FindAllStringSubmatch(text, -1)
	result1 := weixinRE1.FindStringSubmatch(text)
	result2 := weixinRE2.FindStringSubmatch(result1[1])

	fmt.Println(result)
	fmt.Println(result1)
	fmt.Println(result2)
	//ips := strings.Split(res, "\\n")
	//fmt.Println(ips)
	//for _, ip := range ips {
	//	fmt.Println(ip)
	//}
	//fmt.Println(len(strings.Split("a123a234a345a456", "a")))
}
