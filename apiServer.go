package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"ipset-api/cmder"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

//compile for linux with mac
//CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build apiServer.go

func init() {
	logInit()
	gin.SetMode(gin.ReleaseMode)
	mapInit()
	whiteListDstInit()
}

type GetResult struct {
	LocalList []string
}

func whiteListDstInit() {
	resp, err := http.Get(DNSAPI)
	if err != nil {
		log.Println("whiteListDstInit初始化连接DNS服务器失败，请注意启动DNS服务器的go程序：" + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
	if err != nil {
		log.Fatal("whiteListDstInit读取get请求返回值ReadAll error：" + err.Error())
	}

	//反序列化请求结果到getResult中
	var getResult GetResult
	err = json.Unmarshal(body, &getResult)
	if err != nil {
		log.Fatal("whiteListDstInit Unmarshal error：" + err.Error())
	}

	var cmd string
	for _, ip := range getResult.LocalList {
		cmd = "ipset add white_list_dst " + ip
		cmdResult, err := cmder.Exec_shell(cmd)
		if err != nil && !checkStr(cmdResult) {
			log.Fatalf("初始化添加ip：%s到white_list_dst失败，错误信息：%s。"+
				"请将iptables和dns服务器上的go程序重启，添加ip为：", ip, err.Error())
		}
	}
}

//全局变量map[string]int8,保存各组ip信息
//file 为日志文件
var (
	dict sync.Map
	file *os.File
)

func main() {

	defer file.Close()

	//set log default output and prefix
	log.SetOutput(file)
	log.SetPrefix(time.Now().Format("2006/01/02 - 15:04:05") + "-")

	gin.DefaultWriter = io.MultiWriter(file)
	router := gin.Default()
	gin.Logger()

	router.GET("/online-info", getMapInfo)
	router.GET("/group-by-ip", getGroup)
	router.GET("/change-group", changeGroup)
	router.POST("/add-iplist", addIpList)
	//router.POST("/del", deleter)
	router.OPTIONS("/*matchAllOptions", corsOptionsAllow)

	router.Run(":9800")
}

func logInit() {
	dateStr := time.Now().Format(`2006-01-02-15`)
	var (
		logDir  = "/synet/logs"
		logfile = flag.String("logfile",
			logDir+"/go-"+dateStr+".log",
			"set logfile and default is "+logDir+"/go-年-月-日-时.log")
	)
	flag.Parse()
	exist, err := PathExists(logDir)
	if err != nil {
		log.Fatal("判断日志目录是否存在，失败：" + err.Error())
	}
	if !exist {
		if err := os.Mkdir(logDir, os.ModePerm); err != nil {
			log.Fatal("日志目录创建失败：" + err.Error())
		}
	}
	file, err = os.OpenFile(*logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm|os.ModeTemporary)
	if err != nil {
		log.Fatal("日志文件创建失败：" + err.Error())
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

type addListData struct {
	GroupName string
	IpList    []string
}

func addIpList(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	var (
		data        addListData
		statusCode  = 0
		ipList      []string
		successList []string
		groupName   string
	)
	if err := c.ShouldBind(&data); err == nil {
		ipList = data.IpList
		groupName = data.GroupName

		var cmd string
		for _, ip := range ipList {
			cmd = "ipset add " + groupName + " " + ip
			reslut, err := cmder.Exec_shell(cmd)
			if err == nil || checkStr(reslut) {
				successList = append(successList, ip)
			}
		}
		if len(successList) != len(data.IpList) {
			statusCode = 1
		}
		log.Printf("addIpList successList：%s", successList)

		c.JSON(200, gin.H{
			"status":      statusCode,
			"successList": successList,
			"groupName":   groupName,
		})
	}
}

//实现幂等
func checkStr(s string) bool {
	dst := `Element cannot be added to the set: it's already added`
	index := strings.Index(s, dst)
	if index != -1 {
		return true
	}
	return false
}

func corsOptionsAllow(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.JSON(200, nil)
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
	DNSAPI              = "http://172.16.10.91:9800/local-list?kind=startSendData"
	whiteListInMap int8 = 1
	allInMap       int8 = 2
	//当whiteListInRequest == 60010时，表示添加到白名单组，为空表示下线，为其他值表示添加到放通组
	whiteListInRequest string = "60010"
	nullInRequest      string = ""
	//whiteList在linux中的ipset中的组名，执行shell命令需要
	whiteListName string = "white_list_src"
	//all在linux中的ipset中的组名，执行shell命令需要
	allName string = "all"
)

//初始化服务器中的weixin和all组的ip到内存中保存
func mapInit() {
	//首先判断是否存在white_list_src，white_list_dst，all_allow这几个组，如果不存在，创建
	isIpsetCreated()

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

func isIpsetCreated() {
	cmdDst := `ipset list |grep "Name: white_list_dst"`
	cmdSrc := `ipset list |grep "Name: white_list_src"`
	cmdAll := `ipset list |grep "Name: allow_list"`
	dstResult, err := cmder.Exec_shell(cmdDst)
	if err != nil {
		log.Println(err.Error())
	}
	if dstResult == "" {
		log.Println("white_list_dst is not in ipset, now create this set...")
		_, err := cmder.Exec_shell("ipset create white_list_dst hash:net family inet hashsize 4096 maxelem 1000000")
		if err != nil {
			panic(err)
		}
	}
	srcResult, err := cmder.Exec_shell(cmdSrc)
	if err != nil {
		log.Println(err.Error())
	}
	if srcResult == "" {
		log.Println("white_list_src is not in ipset, now create this set...")
		_, err := cmder.Exec_shell("ipset create white_list_src hash:net family inet hashsize 4096 maxelem 1000000")
		if err != nil {
			panic(err)
		}
	}
	allResult, err := cmder.Exec_shell(cmdAll)
	if err != nil {
		log.Println(err.Error())
	}
	if allResult == "" {
		log.Println("allow_list is not in ipset, now create this set...")
		_, err := cmder.Exec_shell("ipset create allow_list hash:net family inet hashsize 4096 maxelem 1000000")
		if err != nil {
			panic(err)
		}
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
