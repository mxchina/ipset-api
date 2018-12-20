package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ipset-api/cmder"
	"strings"
	"sync"
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
	router.GET("/map-info", getMap)
	router.POST("/add", adder)
	router.POST("/del", deleter)
	router.POST("/add-list", addList)
	router.POST("/del-list", deleteList)
	router.POST("/moveSet", moveSeter)
	router.POST("/move-set-list", moveSetList)
	router.OPTIONS("/*matchAllOptions", corsOptionsAllow)
	router.Run(":9800")
}

func corsOptionsAllow(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.JSON(200, nil)
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

type moveSetData struct {
	Ip      string
	SetFrom string
	SetTo   string
}

func moveSeter(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var (
		data moveSetData
		ip   string
	)
	if err := c.ShouldBind(&data); err == nil {
		if len(data.Ip) == 0 {
			ip = c.ClientIP()
		} else {
			ip = data.Ip
		}
		setFrom := strings.ToLower(data.SetFrom)
		setTo := strings.ToLower(data.SetTo)
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
			}
		}
		c.JSON(200, gin.H{
			"code":    code,
			"ip":      ip,
			"setFrom": setFrom,
			"setTo":   setTo,
		})
	}
}

const (
	weixin int8 = 1
	all    int8 = 2
)

//map[string]int8
var dict sync.Map

type ResErr struct {
	Msg string
}

func setMap(ip string, group string) error {
	if group == "weixin" {
		dict.Store(ip, weixin)
		return nil
	} else if group == "all" {
		dict.Store(ip, all)
		return nil
	} else {
		return fmt.Errorf("setMap error：group %q not exist", group)
	}
}

func execAndSetMap(ip, group, action string) error {
	cmd := "ipset " + action + " " + group + " " + ip
	_, err := cmder.Exec_shell(cmd)
	if err != nil {
		return err
	}
	return setMap(ip, group)
}

func execAndDeleteMap(ip, group, action string) error {
	cmd := "ipset " + action + " " + group + " " + ip
	_, err := cmder.Exec_shell(cmd)
	if err != nil {
		return err
	}
	dict.Delete(ip)
	return nil
}

func adder(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var (
		ip     = c.DefaultPostForm("ip", c.ClientIP())
		group  = strings.ToLower(c.PostForm("group"))
		resErr ResErr
		code   = 0
	)
	if group != "weixin" && group != "all" {
		code = 1
		resErr.Msg = fmt.Errorf("group require weixin or all，got %q", group).Error()
	} else {
		groupName, ok := dict.Load(ip)
		if ok {
			// ok表示ip已经在map中
			if groupName == weixin && group == "all" {
				// 从weixin组到all组，1 从weixin组删除ip 2 添加ip到all组
				if err := execAndSetMap(ip, "weixin", "del"); err != nil {
					code = 1
					resErr.Msg = resErr.Msg + err.Error()
				} else {
					if err := execAndSetMap(ip, "all", "add"); err != nil {
						code = 1
						resErr.Msg = resErr.Msg + err.Error()
					}
				}
			}
			if groupName == all && group == "weixin" {
				//从all组到weixin组，1从all组删除ip 2添加ip到weixin组
				if err := execAndSetMap(ip, "all", "del"); err != nil {
					code = 1
					resErr.Msg = resErr.Msg + err.Error()
				} else {
					if err := execAndSetMap(ip, "weixin", "add"); err != nil {
						code = 1
						resErr.Msg = resErr.Msg + err.Error()
					}
				}
			}
		} else {
			// ip不在map中，也就是不在任何组中
			if group == "weixin" {
				if err := execAndSetMap(ip, "weixin", "add"); err != nil {
					code = 1
					resErr.Msg = resErr.Msg + err.Error()
				}
			}
			if group == "all" {
				if err := execAndSetMap(ip, "all", "add"); err != nil {
					code = 1
					resErr.Msg = resErr.Msg + err.Error()
				}
			}
		}
	}

	c.JSON(200, gin.H{
		"code":  code,
		"err":   resErr,
		"ip":    ip,
		"group": group,
	})
}

func deleter(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var (
		ip     = c.DefaultPostForm("ip", c.ClientIP())
		code   = 0
		resErr ResErr
	)
	groupName, ok := dict.Load(ip)
	if ok {
		//判断所在group，然后从其组删除，然后在map中删除
		if groupName == weixin {
			if err := execAndDeleteMap(ip, "weixin", "del"); err != nil {
				code = 1
				resErr.Msg = resErr.Msg + err.Error()
			}
		} else {
			if err := execAndDeleteMap(ip, "all", "del"); err != nil {
				code = 1
				resErr.Msg = resErr.Msg + err.Error()
			}
		}
	} else {
		resErr.Msg = fmt.Errorf("this ip %s is not exist in weixin or all").Error()
	}

	c.JSON(200, gin.H{
		"code":  code,
		"err":   resErr,
		"ip":    ip,
		"group": groupName,
	})
}

func getMap(c *gin.Context) {
	res := ""
	dict.Range(func(key, value interface{}) bool {
		res += fmt.Errorf("%s-->%d\n", key, value).Error()
		return true
	})
	c.String(200, res)
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
