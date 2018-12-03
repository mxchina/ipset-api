//package main
//
//import (
//	"encoding/json"
//	"fmt"
//)
//
//type tokenResult struct {
//	access_token string
//	expires_in int
//}
//
//
//func main() {
//	const str = `{"access_token":"16_RNAcRKxZpQXQRI444Ufj1m7cSaXekxbm3PqAHElBnH6o5mqpx7tvGTYt5-VWHcZhvrTlWqJavUy4ZARppPKcmivBlLV7IgG_L6wg-Jr5DvEGeVYhCHr45xg3oHAZ_cb-gbYbZNAjBi0VvKKyQJXiACABUI","expires_in":7200}`
//	token := tokenResult{}
//	err := json.Unmarshal([]byte(str), &token)
//	fmt.Println(err)
//	fmt.Println(token)
//}
package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var jsonBlob = []byte(`{ "access_token" : "Quoll" ,     "order" : "Dasyuromorphia" }`)
	type Animal struct {
		Access_token string
		Order        string
	}
	var animals Animal
	err := json.Unmarshal(jsonBlob, &animals)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%+v", animals)
}
