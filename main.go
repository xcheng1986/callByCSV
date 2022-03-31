package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/robfig/cron"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//接口返回格式
type (
	ApiResponse struct {
		ErrNo  int         `json:"errNo"`
		ErrStr string      `json:"errstr"`
		Data   interface{} `json:"data"`
	}

	CFG struct {
		url         *string
		csvFilePath *string
		perSecond   *int64
		perCount    *int64
		batch       *int64
		cookies     []*http.Cookie
	}

	csvFileData []map[string]string
)

var (
	data   csvFileData
	config CFG
)

func init() {
	data = csvFileData{}
	url := flag.String("url", "http://api.test-site.com/api/set-data", "URL")
	csvFilePath := flag.String("csvFilePath", "t1.csv", "csvFilePath")
	perSecond := flag.Int64("perSecond", 1, "perSecond")
	perCount := flag.Int64("perCount", 20, "perCount")
	batch := flag.Int64("batch", 0, "batch")
	ips := flag.String("ips", "IPS_1234567890", "ips")
	help := flag.String("h", "", "help")
	flag.Parse()

	if len(*help) > 0 {
		fmt.Println("use [-url=$url] [-csvFilePath=$csvFilePath] [-perSecond=$perSecond] [-perCount=&perCount] [-batch=&batch] [-ips=&ips]")
		os.Exit(0)
	}

	config = CFG{
		url:         url,
		csvFilePath: csvFilePath,
		perSecond:   perSecond,
		perCount:    perCount,
		batch:       batch,
		cookies: []*http.Cookie{
			{
				Name:  "ZYBKey",
				Value: *ips,
			},
		},
	}
}

func main() {
	err := readCsv(*(config.csvFilePath))
	if err != nil {
		log.Fatalf(err.Error())
	}

	allDataNum, _ := strconv.ParseInt(strconv.Itoa(len(data)), 10, 64)

	var (
		myCron     = cron.New()                                      //
		curDataNum = allDataNum - (*config.batch)*(*config.perCount) //当前数据量
		m          *sync.Mutex
	)
	m = new(sync.Mutex)

	myCron.Schedule(cron.Every(time.Second*time.Duration(*config.perSecond)), cron.FuncJob(func() {
		defer func() { *config.batch++ }()

		currentList := getDataList(*config.batch, *config.perCount)
		if len(currentList) == 0 {
			myCron.Stop()
			return
		}

		wg2 := sync.WaitGroup{}
		wg2.Add(len(currentList))
		for _, li := range currentList {

			m.Lock()
			curDataNum--
			m.Unlock()

			go func(li map[string]string, curDataNum int64) {
				sleepNum := time.Duration(0.9*1000*(*config.perSecond)/(*config.perCount)) * time.Millisecond

				var curURLArr []string
				for key, val := range li {
					curURLArr = append(curURLArr, strTrim(key)+"="+strTrim(val))
				}

				curURL := *config.url + "?" + strings.Join(curURLArr, "&")

				resp, err := callURL(curURL)
				m.Lock()
				if err != nil {
					fmt.Println("【error】", *config.batch, curDataNum, li, curURL, resp)
				} else {
					fmt.Println("【success】", *config.batch, curDataNum, li, resp)
				}
				m.Unlock()

				time.Sleep(sleepNum)
				wg2.Done()
			}(li, curDataNum)
		}

	}))
	myCron.Run()
}

/**
 * strTrim
 */
func strTrim(str string) string {
	trimStr := []string{
		" ",
		"\r",
		"\n",
		"\r\n",
		"\t",
		"\b",
		"\uFEFF",
	}

	for _, tStr := range trimStr {
		str = strings.Replace(str, tStr, "", -1)
	}

	return str
}

/**
 * 读取切片
 */
func getDataList(batch int64, PerCount int64) (list []map[string]string) {
	start := batch * PerCount
	end := (batch + 1) * PerCount

	allDataNum, _ := strconv.ParseInt(strconv.Itoa(len(data)), 10, 64)
	if start > allDataNum {
		return
	}

	if end > allDataNum {
		end = allDataNum
	}

	list = data[start:end]
	return
}

/**
 * 读取CSV文件
 */
func readCsv(csvFilePath string) (err error) {
	fs, err := os.Open(csvFilePath)
	if err != nil {
		return
	}
	defer fs.Close()

	var titles map[int]string
	titles = map[int]string{}

	r := csv.NewReader(fs)
	line := 0
	for {
		line++

		row, err := r.Read()
		if err != nil && err != io.EOF {
			log.Fatalf("can not read, err is %+v", err)
			return err
		}

		if err == io.EOF {
			break
		}

		var rowMap map[string]string
		rowMap = map[string]string{}

		if line == 1 {
			for idx, v := range row {
				titles[idx] = v
			}
		} else {
			for idx, v := range row {
				rowMap[titles[idx]] = v
			}

			data = append(data, rowMap)
		}
	}

	return nil
}

/**
 * 对[php-server]进行回调
 * @param url string
 * @param postString string key1=val1&key2=val2&key3=val3
 */
func callURL(url string) (data ApiResponse, err error) {
	var req *http.Request
	var resp *http.Response
	var httpClient = &http.Client{}

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	for _, k := range config.cookies {
		req.AddCookie(k)
	}
	resp, err = httpClient.Do(req)
	if err != nil {
		err = errors.New("RequestError:" + err.Error() + " url:" + url)
		return
	} else {
		defer func() { _ = resp.Body.Close() }()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = errors.New("RequestError:" + err.Error() + " url:" + url)
		return
	}

	var R ApiResponse
	err2 := json.Unmarshal(body, &R)
	if err2 != nil {
		err = errors.New("JsonUnmarshalError:" + string(body) + " url:" + url)
		return
	}

	if R.ErrNo == 0 {
		return R, nil
	} else {
		err = errors.New(fmt.Sprintf("[code:%d]", R.ErrNo) + " : " + R.ErrStr + " url:" + url)
		return
	}
}
