package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"ipset-api/cmder"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//定义结构ipset同步器
type ipsetSyncer struct {
	localList  []string
	toSendList []string
	diffList   []string
	lock       sync.Mutex
}

const (
	iptablesIP    = "172.16.10.80"
	iptablesPort  = "9800"
	postURL       = "http://" + iptablesIP + ":" + iptablesPort + "/add-iplist"
	postGroupName = "white_list_dst"
	getListKind   = "startSendData"
)

var (
	//全局变量
	ipsetSync         ipsetSyncer
	whiteListDstRE, _ = createWhiteListDstRE()
	file              *os.File
	logFlag           = [5]bool{true, true, true, true, true}
	startFlag         bool
)

func init() {
	gin.SetMode(gin.ReleaseMode)
	logInit()
	ipsetSync.listInit()
	ipsetSync.sendlistToIptables()
	if startFlag == true {
		go startMonitor()
	}
}

// 循环监视本地ipset list，如果有更新的ip，则发送到iptables服务器上去。
func startMonitor() {
	log.Println("开启本地ipset list监视")
	for {
		ipsetSync.syncIPListFromIpset()
		ipsetSync.sendlistToIptables()
		//time.Sleep(time.Second)
	}
}

func contains(src string, dst []string) bool {
	for _, i := range dst {
		if i == src {
			return true
		}
	}
	return false
}

func (i *ipsetSyncer) syncIPListFromIpset() {
	//获取ipset中的white_list_dst集合ip列表
	ipList := getIpsetList(whiteListDstRE)
	//和localList做对比，并将新的ip保存到diffList
	//如果diffList不为空，将diffList中的ip更新到localList，同时更新到toSendList。然后将toSendList发送到iptables服务器上完成同步。
	i.lock.Lock()
	defer i.lock.Unlock()
	for _, ip := range ipList {
		if ok := contains(ip, i.localList); !ok {
			i.diffList = append(i.diffList, ip)
		}
	}

	for _, ip := range i.diffList {
		i.localList = append(i.localList, ip)
		i.toSendList = append(i.toSendList, ip)
	}

	//清空diffList
	if len(i.diffList) != 0 {
		i.diffList = []string{}
	}
}

func (i *ipsetSyncer) sendlistToIptables() {
	i.lock.Lock()
	defer i.lock.Unlock()
	if len(i.toSendList) != 0 {
		var (
			url      = postURL
			sendData = postSendData{
				GroupName: postGroupName,
				IpList:    i.toSendList,
			}
		)
		bytesData, err := json.Marshal(sendData)
		if err != nil {
			//如果失败，只记录一条日志
			if logFlag[0] == true {
				log.Println("Marshal失败" + err.Error())
				logFlag[0] = false
			}
			return
		} else {
			logFlag[0] = true
		}
		resp, err := http.Post(url, "application/json", bytes.NewReader(bytesData))
		if err != nil {
			// 如果对端iptables上的apiServer没有起来，只记录一条日志，
			if logFlag[1] == true {
				log.Println("postErr：" + err.Error())
				logFlag[1] = false
			}
			return
		} else {
			logFlag[1] = true
			//如果对端iptables上的apiServer起来了，那么设置start标志位为true，开启监视
			startFlag = true
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if logFlag[2] == true {
				log.Println("StatusCode not 200：" + strconv.Itoa(resp.StatusCode))
				logFlag[2] = false
			}
			return
		} else {
			logFlag[2] = true
		}

		body, err := ioutil.ReadAll(bufio.NewReader(resp.Body))
		if err != nil {
			if logFlag[3] == true {
				log.Println("读取post请求返回值ReadAll error：" + err.Error())
				logFlag[3] = false
			}
			return
		} else {
			logFlag[3] = true
		}

		//反序列化请求结果到postResult中
		var postResult PostResult
		err = json.Unmarshal(body, &postResult)
		if err != nil {
			if logFlag[4] == true {
				log.Println("Unmarshal error：" + err.Error())
				logFlag[4] = false
			}
			return
		} else {
			logFlag[4] = true
		}
		// 2 将postResult.SuccessList中的ip从本地toSendList中删除
		for _, ip := range postResult.SuccessList {
			for j := range i.toSendList {
				if i.toSendList[j] == ip {
					i.toSendList = append(i.toSendList[:j], i.toSendList[j+1:]...)
					//一定不要忘了加这个break，否则就是在循环中改变循环对象
					break
				}
			}
		}
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

func logInit() {
	dateStr := time.Now().Format(`2006-01-02-15`)
	var (
		logDir  = "/synet/logs"
		logfile = flag.String("logfile",
			logDir+"/go-"+dateStr+".log",
			"set logfile and default is "+logDir+"/go-年-月-日-时.log")
	)
	flag.Parse()

	//先创建日志目录，判断
	exist, err := PathExists(logDir)
	if err != nil {
		log.Fatal("判断日志目录是否存在，失败")
	}

	if !exist {
		if err := os.Mkdir(logDir, os.ModePerm); err != nil {
			log.Fatal("日志目录创建失败")
		}
	}

	file, err = os.OpenFile(*logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm|os.ModeTemporary)
	if err != nil {
		log.Fatal("日志文件创建失败")
	}

	//set log default output and prefix
	log.SetOutput(file)
	log.SetPrefix(time.Now().Format("2006/01/02 - 15:04:05") + "-")
}

func main() {

	defer file.Close()

	gin.DefaultWriter = io.MultiWriter(file)
	router := gin.Default()
	gin.Logger()

	router.GET("/send-list", getSendList)
	router.GET("/local-list", getLocalList)
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
	//router.POST("/del", deleter)
	router.OPTIONS("/*matchAllOptions", corsOptionsAllow)

	router.Run(":9800")
}

type PostResult struct {
	StatusCode  int
	SuccessList []string
	GroupName   string
}
type postSendData struct {
	GroupName string
	IpList    []string
}

func createWhiteListDstRE() (*regexp.Regexp, error) {
	var (
		//注意[\s\S]才能匹配任意字符，.匹配不到\n换行符
		whiteListDstRE = regexp.MustCompile(`Name: white_list_dst[\s\S]*?Members:\n([\s\S]*?)\n\nName`)
	)
	//先判断weixin和all两个组，在服务器上ipset list命令后，所显示的位置。
	//如果white_list_dst组在最后，那么获取该组IP列表的正则表达式不一样。go好像不支持正则表达式(?:)
	groupList, err := cmder.Exec_shell("ipset list|grep Name")
	if err == nil {
		groupList = strings.TrimSpace(groupList)
		split := strings.Split(groupList, "\n")
		if strings.HasSuffix(split[len(split)-1], "white_list_dst") {
			whiteListDstRE = regexp.MustCompile(`Name: white_list_dst[\s\S]*?Members:\n([\s\S]*)\n`)
		}
	} else {
		log.Printf("初始化失败：%s\n", groupList)
		log.Fatal(err)
	}
	return whiteListDstRE, err
}

//初始化服务器中的white_list_dst组的ip到内存中保存，type为slice，变量名localList
func (i *ipsetSyncer) listInit() {
	i.lock.Lock()
	defer i.lock.Unlock()
	ipList := getIpsetList(whiteListDstRE)
	//复制ipList到localList
	for _, ip := range ipList {
		i.localList = append(i.localList, ip)
	}

	//复制localList到toSendList
	for _, ip := range i.localList {
		i.toSendList = append(i.toSendList, ip)
	}
}

func getIpsetList(whiteListDstRE *regexp.Regexp) []string {

	//log.Println(fmt.Sprintf("whiteListRE:%s", whiteListRE.String()))
	//获取white_list_dst组中的IP列表
	text, err := cmder.Exec_shell("ipset list")
	if err != nil {
		log.Printf("初始化失败：%s", text)
		log.Fatal(err)
	}
	whiteList := whiteListDstRE.FindStringSubmatch(text)
	var localList []string
	if len(whiteList) >= 2 {
		localList = strings.Split(whiteList[1], "\n")
	} else {
		localList = []string{}
	}

	return localList
}

func getSendList(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	ipsetSync.lock.Lock()
	defer ipsetSync.lock.Unlock()
	c.JSON(200, gin.H{
		"toSendList": ipsetSync.toSendList,
	})
}

func getLocalList(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	kind := c.Query("kind")
	ip := c.ClientIP()
	ipsetSync.lock.Lock()
	defer ipsetSync.lock.Unlock()
	c.JSON(200, gin.H{
		"localList": ipsetSync.localList,
	})
	if kind == getListKind && ip == iptablesIP && startFlag == false {
		// 如果接收到iptables服务器发送过来的请求，且QueryString中带有kind=startSendData，且初始化时没有开启监视
		// 就开个goroutine一直循环监视本地ipset list，发送最新的white_list_dst中的ip过去
		go startMonitor()
	}
}

func corsOptionsAllow(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.JSON(200, nil)
}
