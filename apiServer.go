package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go_firewall/cmder"
	"strings"
)

/*
POST /post?id=1234&page=1 HTTP/1.1
Content-Type: application/x-www-form-urlencoded

name=manu&message=this_is_great
*/

func main() {
	router := gin.Default()

	router.GET("/setFromIP", getSeter)
	router.GET("/membersFromSet", getMembers)
	router.POST("/add", adder)
	router.POST("/del", deleter)
	router.POST("/add-list", addList)
	router.POST("/del-list", deleteList)
	router.POST("/moveSet", moveSeter)
	router.POST("/move-set-list", moveSetList)
	router.Run(":9800")
}

func getMembers(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
}

func getSeter(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ip := c.DefaultQuery("ip", c.ClientIP())
	authCMD := "ipset test Auth " + ip
	permitCMD := "ipset test Permit " + ip
	setName := ""
	if _, err := cmder.Exec_shell(authCMD); err == nil {
		//ip in Auth
		setName = "auth"
	} else {
		if _, err := cmder.Exec_shell(permitCMD); err == nil {
			//ip in Permit
			setName = "permit"
		} else {
			// ip not in auth,permit
			setName = "none"
		}
	}
	c.JSON(200, gin.H{
		"code":    0,
		"ip":      ip,
		"setName": setName,
	})
}

func moveSeter(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ip := c.DefaultPostForm("ip", c.ClientIP())
	setFrom := strings.ToLower(c.PostForm("setFrom"))
	setTo := strings.ToLower(c.PostForm("setTo"))
	var (
		cmd  string
		code = 0
	)
	if setFrom == "none" && setTo == "auth" {
		cmd = "ipset add Auth " + ip
	}
	if setFrom == "auth" && setTo == "permit" {
		cmd = "ipset del Auth " + ip + " && ipset add Permit " + ip
	}
	if setFrom == "auth" && setTo == "none" {
		cmd = "ipset del Auth " + ip
	}
	if setFrom == "permit" && setTo == "none" {
		cmd = "ipset del Permit " + ip
	}

	if cmd == "" {
		code = 1
	} else {
		_, err := cmder.Exec_shell(cmd)
		if err != nil {
			code = 1
			fmt.Println(err)
		}
	}
	c.JSON(200, gin.H{
		"code":    code,
		"ip":      ip,
		"setFrom": setFrom,
		"setTo":   setTo,
	})
}

func deleter(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ip := c.DefaultPostForm("ip", c.ClientIP())
	setName := c.PostForm("setName")
	cmd := "ipset del " + setName + " " + ip
	_, err := cmder.Exec_shell(cmd)
	responeseWrap(err, c, ip, setName)
}

func adder(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	//id := c.DefaultQuery("qip","")
	//page := c.DefaultQuery("qSetName", "0")
	ip := c.DefaultPostForm("ip", c.ClientIP())
	setName := c.PostForm("setName")
	cmd := "ipset add " + setName + " " + ip
	_, err := cmder.Exec_shell(cmd)
	responeseWrap(err, c, ip, setName)
}

func addList(c *gin.Context) {
	ListAdderAndDeleter(c, "add")
}

func deleteList(c *gin.Context) {
	ListAdderAndDeleter(c, "del")
}

type ListAdderAndDeleterData struct {
	AuthIpList   []string
	PermitIpList []string
}

type ResponeseSetResponese struct {
	Code   int
	IpList []string
}

func ListAdderAndDeleter(c *gin.Context, action string) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var data ListAdderAndDeleterData
	if c.ShouldBind(&data) == nil {
		var (
			// cmd初始化，因为第一个命令前有个&&，所以需要前置一个无意义的命令
			authCmd    = "takePlace=0"
			permitCmd  = "takePlace=0"
			authCode   = 0
			permitCode = 0
		)
		for _, ip := range data.AuthIpList {
			authCmd = authCmd + "&&ipset " + action + " Auth " + ip
		}
		if authCmd != "takePlace=0" {
			_, err := cmder.Exec_shell(authCmd)
			if err != nil {
				authCode = 1
			}
		}
		for _, ip := range data.PermitIpList {
			permitCmd = permitCmd + "&&ipset " + action + " Permit " + ip
		}
		if permitCmd != "takePlace=0" {
			_, err := cmder.Exec_shell(permitCmd)
			if err != nil {
				permitCode = 1
			}
		}
		c.JSON(200, gin.H{
			//"auth": ResponeseSetResponese{
			//	Code:   authCode,
			//	IpList: data.AuthIpList,
			//},
			"authCode":   authCode,
			"permitCode": permitCode,
		})
	}
}

/*
{
"SetFrom": "auth",
"SetTo": "permit",
"IpList": ["1.1.1.1","4.4.4.4"]
}
*/

type MoveSetListData struct {
	SetFrom string
	SetTo   string
	IpList  []string
}

func moveSetList(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var data MoveSetListData
	if c.ShouldBind(&data) == nil {
		var (
			cmd  = "takePlace=0"
			code = 0
		)
		if strings.ToLower(data.SetFrom) == "none" && strings.ToLower(data.SetTo) == "auth" {
			for _, ip := range data.IpList {
				cmd = cmd + "&&ipset add Auth " + ip
			}
			if cmd != "takePlace=0" {
				_, err := cmder.Exec_shell(cmd)
				if err != nil {
					code = 1
				}
			}
		}
		if strings.ToLower(data.SetFrom) == "auth" && strings.ToLower(data.SetTo) == "permit" {
			for _, ip := range data.IpList {
				cmd = cmd + "&&ipset del Auth " + ip + " && ipset add Permit " + ip
			}
			if cmd != "takePlace=0" {
				_, err := cmder.Exec_shell(cmd)
				if err != nil {
					code = 1
				}
			}
		}
		if strings.ToLower(data.SetFrom) == "auth" && strings.ToLower(data.SetTo) == "none" {
			for _, ip := range data.IpList {
				cmd = cmd + "&&ipset del Auth " + ip
			}
			if cmd != "takePlace=0" {
				_, err := cmder.Exec_shell(cmd)
				if err != nil {
					code = 1
				}
			}
		}
		if strings.ToLower(data.SetFrom) == "permit" && strings.ToLower(data.SetTo) == "none" {
			for _, ip := range data.IpList {
				cmd = cmd + "&&ipset del Permit " + ip
			}
			if cmd != "takePlace=0" {
				_, err := cmder.Exec_shell(cmd)
				if err != nil {
					code = 1
				}
			}
		}
		c.JSON(200, gin.H{
			"code":    code,
			"setFrom": data.SetFrom,
			"setTo":   data.SetTo,
		})
	}
}

func responeseWrap(err error, c *gin.Context, ip string, setName string) {
	code := 0
	if err != nil {
		code = 1
		fmt.Println(err)
	}
	c.JSON(200, gin.H{
		"code":    code,
		"ip":      ip,
		"setName": setName,
	})
}
