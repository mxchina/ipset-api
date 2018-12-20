package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"ipset-api/cmder"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

func init() {
	mapInit()

	// TODO:
	// go checkSync()
}

//compile for linux with mac
//CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build apiServer.go

func main() {
	var logfile = flag.String("logfile", "/synet/gin.log", "set logfile and default is /synet/gin.log")
	flag.Parse()

	f, _ := os.Create(*logfile)
	defer f.Close()

	//set log default output and prefix
	log.SetOutput(f)
	log.SetPrefix(time.Now().Format("2006/01/02 - 15:04:05") + "-")

	gin.DefaultWriter = io.MultiWriter(f)
	router := gin.Default()
	gin.Logger()

	router.GET("/membersFromSet", getMembers)
	router.GET("/online-info", getMapInfo)
	router.GET("/group-by-ip", getGroup)
	router.GET("/change-group", changeGroup)
	//router.POST("/del", deleter)
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
	var (
		setName = c.Query("group")
		code    = 0
		cmd     = "ipset list"
		RE      *regexp.Regexp
	)
	text, err := cmder.Exec_shell(cmd)
	switch strings.ToLower(setName) {
	case "weixin":
		RE = regexp.MustCompile(`Name: weixin.*Members:\\n(.*)\\n{\nName}.`)
	case "auth":
		RE = regexp.MustCompile(`Name: weixin.*Members:\\n(.*)\\n{\nName}.`)
	case "permit":
		RE = regexp.MustCompile(`Name: Permit.*Members:\\n(.*?)\\n\\nName: Weixin`)
	}
	if err == nil {
		result := RE.FindStringSubmatch(text)
		var ipList []string
		if len(result) >= 2 {
			ipList = strings.Split(result[1], "\\n")
		} else {
			ipList = []string{}
		}

		c.JSON(200, gin.H{
			"code":    code,
			"group":   setName,
			"members": ipList,
		})
	}
}

func getGroup(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ip := c.DefaultQuery("ip", c.ClientIP())
	var (
		group string
	)
	value, ok := dict.Load(ip)
	if ok {
		//在weixin 或 all中
		if value == whiteListInMap {
			group = "white_list_client"
		} else {
			group = "all"
		}
	} else {
		group = "none"
	}
	c.JSON(200, gin.H{
		"code":  0,
		"ip":    ip,
		"group": group,
	})
}

const (
	whiteListInMap     int8   = 1
	allInMap           int8   = 2
	whiteListInRequest string = "60010"
	nullInRequest      string = ""
	whiteListName      string = "white_list_src" //whiteList在linux中的ipset中的组名，执行shell命令需要
	allName            string = "all"            //all在linux中的ipset中的组名，执行shell命令需要
)

//全局变量map[string]int8,保存各组ip信息
var dict sync.Map

//初始化服务器中的weixin和all组的ip到内存中保存
func mapInit() {
	var (
		//注意[\s\S]才能匹配任意字符，.匹配不到\n换行符
		whiteListRE = regexp.MustCompile(`Name: white_list_src[\s\S]*?Members:\n([\s\S]*?)\n\nName`)
		allRE       = regexp.MustCompile(`Name: all[\s\S]*?Members:\n([\s\S]*?)\n\nName`)
	)
	//先判断weixin和all两个组，在服务器上ipset list命令后，所显示的位置。
	//如果有一个组在最后，那么获取该组IP列表的正则表达式不一样。go好像不支持正则表达式(?:)
	groupList, err := cmder.Exec_shell("ipset list|grep Name")
	if err == nil {
		groupList = strings.TrimSpace(groupList)
		split := strings.Split(groupList, "\n")
		if strings.HasSuffix(split[len(split)-1], "white_list_src") {
			whiteListRE = regexp.MustCompile(`Name: white_list_src[\s\S]*?Members:\n([\s\S]*)\n`)
		}
		if strings.HasSuffix(split[len(split)-1], "all") {
			allRE = regexp.MustCompile(`Name: all[\s\S]*?Members:\n([\s\S]*)\n`)
		}
	} else {
		log.Printf("初始化失败：%s", groupList)
		log.Fatal(err)
	}
	//log.Println(fmt.Sprintf("whiteListRE:%s", whiteListRE.String()))
	//log.Println(fmt.Sprintf("allListRE:%s", allRE.String()))

	//获取white_list_client和all两个组中的IP列表
	text, err := cmder.Exec_shell("ipset list")
	if err != nil {
		log.Printf("初始化失败：%s", text)
		log.Fatal(err)
	}
	whiteList := whiteListRE.FindStringSubmatch(text)
	allList := allRE.FindStringSubmatch(text)
	//log.Println(fmt.Sprintf("whiteList:%s", whiteList))
	//log.Println(fmt.Sprintf("allList:%s", allList))
	var (
		whiteIpList []string
		allIpList   []string
	)

	if len(whiteList) >= 2 {
		whiteIpList = strings.Split(whiteList[1], "\n")
	} else {
		whiteIpList = []string{}
	}

	if len(allList) >= 2 {
		allIpList = strings.Split(allList[1], "\n")
	} else {
		allIpList = []string{}
	}
	//log.Println(fmt.Sprintf("whiteIpList:%s", whiteIpList))
	//log.Println(fmt.Sprintf("allIpList:%s", allIpList))

	//遍历添加两个组中的ip到map中，同步服务器ipset数据，完成初始化
	for _, ip := range whiteIpList {
		dict.Store(ip, whiteListInMap)
	}
	for _, ip := range allIpList {
		dict.Store(ip, allInMap)
	}

}

func setMap(ip string, group string) error {
	//group == "all" || group == "white_list_client"
	if group == "all" {
		dict.Store(ip, allInMap)
		return nil
	} else {
		dict.Store(ip, whiteListInMap)
		return nil
	}
}

func execAndSetMap(ip, group, action string) error {
	cmd := "ipset " + action + " " + group + " " + ip
	cmdOut, err := cmder.Exec_shell(cmd)
	if err != nil {
		return fmt.Errorf(cmdOut)
	}
	return setMap(ip, group)
}

func execAndDeleteMap(ip, group, action string) error {
	cmd := "ipset " + action + " " + group + " " + ip
	cmdOut, err := cmder.Exec_shell(cmd)
	if err != nil {
		return fmt.Errorf(cmdOut)
	}
	dict.Delete(ip)
	return nil
}

func changeGroup(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var (
		kind          = strings.ToLower(c.Query("kind"))
		userIp        = c.DefaultQuery("userIp", c.ClientIP())
		userGroupName = strings.ToLower(c.DefaultQuery("userGroupName", nullInRequest))
		resErr        string //错误收集，可以打印到日志，也可以返回到responese
		statusCode    = 1    //1表示成功，0表示失败
	)
	groupNameInMap, ok := dict.Load(userIp)
	if kind == "prelogin" && userGroupName != nullInRequest {
		if ok {
			// ok表示ip已经在map中
			if groupNameInMap == whiteListInMap && userGroupName != whiteListInRequest {
				// userGroupName != null 表示要么放通，要么到白名单
				// 从whiteListInMap组到all组，1 从whiteList组删除ip 2 添加ip到all组
				if err := execAndSetMap(userIp, whiteListName, "del"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				} else {
					if err := execAndSetMap(userIp, allName, "add"); err != nil {
						statusCode = 0
						resErr = resErr + err.Error()
					}
				}
			}
			if groupNameInMap != whiteListInMap && userGroupName == whiteListInRequest {
				//从all组到whiteList组，1从all组删除ip 2添加ip到whiteList组
				if err := execAndSetMap(userIp, allName, "del"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				} else {
					if err := execAndSetMap(userIp, whiteListName, "add"); err != nil {
						statusCode = 0
						resErr = resErr + err.Error()
					}
				}
			}
		} else {
			// ip不在map中，也就是不在任何组中
			if userGroupName == whiteListInRequest {
				if err := execAndSetMap(userIp, whiteListName, "add"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				}
			}
			if userGroupName != whiteListInRequest {
				if err := execAndSetMap(userIp, allName, "add"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				}
			}
		}

	} else if kind == "logout" || userGroupName == nullInRequest {
		if ok {
			//判断所在group，然后从其组删除，然后在map中删除
			if groupNameInMap == whiteListInMap {
				//在白名单组中，从该组中删掉该ip，然后在map中删除
				if err := execAndDeleteMap(userIp, whiteListName, "del"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				}
			} else {
				//在all组中，从all组删掉该ip，然后在map中删除
				if err := execAndDeleteMap(userIp, allName, "del"); err != nil {
					statusCode = 0
					resErr = resErr + err.Error()
				}
			}
		} else {
			resErr = fmt.Sprintf("this ip %s is not exist in whiteList or all group", userIp)
			statusCode = 0
		}
	} else {
		//kind err or userGroupName err
		resErr = fmt.Sprintf("parameter kind: %s not match parameter userGroupName: %s", kind, userGroupName)
		statusCode = 0
	}

	if resErr != "" {
		log.Println(resErr)
	}

	c.String(200, fmt.Sprintf("%d:msg", statusCode))
}

func getMapInfo(c *gin.Context) {
	var (
		whiteListCount int
		allCount       int
		res            string
	)
	dict.Range(func(key, value interface{}) bool {
		res += fmt.Sprintf("%s-->%s\n", key, group2str(value))
		if value.(int8) == whiteListInMap {
			whiteListCount += 1
		} else {
			allCount += 1
		}
		return true
	})
	c.String(200, fmt.Sprintf(
		"info:\n%s\nwhiteListCount:%d\nallCount:%d\n",
		res, whiteListCount, allCount))
}

func group2str(tmpValue interface{}) interface{} {
	if tmpValue == whiteListInMap {
		return "whiteList"
	} else {
		return "all"
	}
}
