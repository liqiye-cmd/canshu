package main

import (
	"bufio"
	"os"
	"flag"
	"sync"
	"math/rand"
	"regexp"
	"net/http"
	"io/ioutil"
	"fmt"
	"io"
	"strings"
	"github.com/logrusorgru/aurora"
)

var au aurora.Aurora
var factors map[string]bool
var details bool

func init() {
	au = aurora.NewAurora(true)
}

func main() {
	var urls []string	
	
	flag.BoolVar(&details, "v", false, "输出详情")

	var path string
	flag.StringVar(&path, "f", "./params.txt", "设置参数字典")

	var url string
	flag.StringVar(&url, "u", "", "目标网址")

	flag.Parse()



	if url != "" {
		urls = []string{url}

	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			urls = append(urls, sc.Text())
		}

		if err := sc.Err(); err != nil && details {
			fmt.Fprintf(os.Stderr, " 从输入流读取参数失败: %s\n ", err)
		}
	}


	paramFromFile, err := ReadLine(path)

	if err != nil && details{
		fmt.Fprintf(os.Stderr, " 从参数字典读取参数失败: %s\n ", au.Red(err))
	}



	for _, url := range urls {
		foundParamsTemp := make(chan string)
	
		foundParams := make(chan string)
	
	
		if details {
			fmt.Println(au.Magenta("[~] 分析网页内容"))
			fmt.Println(au.Yellow(url))
		}
		
		firstResponse, _, err := httpGet(url)
	
		if err != nil && details{
			fmt.Fprintf(os.Stderr, " http请求失败 : %s\n ", au.Red(err))
		}
		
		originalFuzz := RandomString(6)
		originalResponse, originalCode, err := httpGet(url + "?" + RandomString(8) + "=" + originalFuzz)
		
		if err != nil && details{
			fmt.Fprintf(os.Stderr, " http请求失败 : %s\n ", au.Red(err))
		}
		
		reflections := strings.Count(string(originalResponse), originalFuzz)
		if details {
			fmt.Printf("%s %d\n", au.Magenta("随机参数反射次数:"), au.Green(reflections))
			fmt.Printf("%s %d\n", au.Magenta("响应状态码:"), au.Green(originalCode))
		}
		newLength := len(originalResponse)
		plainText := removeTags(string(originalResponse))
		plainTextLength := len(plainText)
		if details {
			fmt.Printf("%s %d\n", au.Magenta("内容长度:"), au.Green(newLength))
			fmt.Printf("%s %d\n", au.Magenta("去除标签以后的内容长度:"),  au.Green(plainTextLength))
		}
		factors = make(map[string]bool)
		factors["sameHTML"] = false
		factors["samePlainText"] = false
		
		if len(firstResponse) == newLength {
			factors["sameHTML"] = true
		}

		if len(removeTags(string(firstResponse))) == plainTextLength {
			factors["samePlainText"] = true
		}
	
		
		paramFromHtml := heuristic(firstResponse)
		
		paramSlice := append(paramFromHtml, paramFromFile...)
	
		paramCheck := splitArray(paramSlice, int64(len(paramSlice)/100))

		var foundParamsTempWG sync.WaitGroup
		
		for _, params := range paramCheck {
			foundParamsTempWG.Add(1)
			go func(params []string) {
				temp := quickBruter(params, originalResponse, originalCode, reflections, url)
				if len(temp) != 0 {
					for _, t := range temp {
  						foundParamsTemp <- t
					}
				}
				foundParamsTempWG.Done()
			}(params) 
			
		}
		
		go func() {
			foundParamsTempWG.Wait()
			close(foundParamsTemp)
		}()
		
		
		var foundParamsWG sync.WaitGroup
	
		
		for param := range foundParamsTemp {
			params := []string{param}
			foundParamsWG.Add(1)
			go func(params []string) {
				exists := quickBruter(params, originalResponse, originalCode, reflections, url)
				if len(exists) != 0 {
					foundParams <- exists[0]
				}
				foundParamsWG.Done()
			}(params) 
		}

		go func() {
			foundParamsWG.Wait()
			close(foundParams)
		}()
		
		
		allParams := []string{}
		for each := range foundParams {
			if details {
				fmt.Printf("%s %s\n", au.Magenta("找到的参数"), au.Red(each))
			}
			allParams = append(allParams, each)
		}
		
		out, _ := joiner(allParams)
		if details {
			fmt.Println(au.Red(url + "?" + out))
		}else {
			fmt.Println(url + "?" + out)
		}
			
		if len(allParams) == 0 {
			if details {
				fmt.Println(au.Yellow("没有找到参数"))
			}
		}
		
		if details {
			fmt.Println(au.Green("================================================================================================================================================\n"))
		}
	}
}




func ReadLine(fileName string) ([]string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewReader(f)
	var result []string
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil {
			if err == io.EOF {
				return result, nil
			}
			return nil, err
		}
		result = append(result, line)
	}
	return result, nil

}

func httpGet(url string) ([]byte, int, error) {
	res, err := http.Get(url)
	
	if nil != res {
		defer res.Body.Close()
	}
	
	if err != nil {
		return []byte{}, 0, err
	}

	raw, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return []byte{}, 0, err
	}

	return raw, res.StatusCode, nil
}


func splitArray(arr []string, num int64) [][]string {
	max := int64(len(arr))
	if max < num {
		return nil
	}
	var segmens = make([][]string, 0)
	quantity := max / num
	end := int64(1)
	for i := int64(0); i <= num; i++ {
		qu := i * quantity
		if i != num {
			segmens = append(segmens, arr[i-1+end:qu])
		} else {
			segmens = append(segmens, arr[i-1+end:])
		}
		end = qu - i
	}
	return segmens[1:]
}


var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")


func RandomString(n int) string {
	s := make([]rune, n)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}


func removeTags(html string) string {
	re, _ := regexp.Compile(`(?s)<.*?>`)
	return re.ReplaceAllString(html, "")
}


func heuristic(response []byte) []string {
	result := []string{}
	out := []string{}
	printed := make(map[string]bool)
	jsvars := regexp.MustCompile(`var(\s+)?([a-zA-Z0-9\-\_]+)(\s+)?=`).FindAllStringSubmatch(string(response), -1)
	for _, jsvar := range jsvars {
		out = append(out, jsvar[2])
	}
	elements := regexp.MustCompile(`(?:name|id)(\s+)?=(\s+)?["']?([a-zA-Z0-9\-\_]+)["\']?`).FindAllStringSubmatch(string(response), -1)
	for _, element := range elements {
		out = append(out, element[3])
	}
	dataattributes := regexp.MustCompile(`data-([a-zA-Z0-9\-\_]+)`).FindAllStringSubmatch(string(response), -1)
	for _, dataattribute := range dataattributes {
		out = append(out, dataattribute[1])
	}
	for _, n := range out {
		if _, ok := printed[n]; !ok {
			result = append(result, n)
			printed[n] = true
		}
	}
	return result
}

func quickBruter(params []string, originalResponse []byte, originalCode int, reflections int, url string) []string {
	joined, joinedm := joiner(params)
	newResponse, newCode, err := httpGet(url + "?" + joined)
	if err != nil && details{
		fmt.Fprintf(os.Stderr, " 失败的http请求: %s\n ", au.Red(err))
	}
	
	if newCode == 429 && details{
		fmt.Println(au.Red("注意，有速率限制！！！！！！"))
		return params
	}
	if newCode != originalCode {
		return params
	} else if factors["sameHTML"] && len(newResponse) != len(originalResponse) {
		return params
	} else if factors["samePlainText"] && len(removeTags(string(originalResponse))) != len(removeTags(string(newResponse))) {
		return params
	} else {
		for _, value := range joinedm {
			if strings.Count(string(newResponse), value) != reflections {
				return params
			}
		}
	}
	return []string{}
}

func joiner(params []string) (string, map[string]string) {
	out := ""
	outm := map[string]string{}
	for _, p := range params {
		t := RandomString(6)
		out += "&" + p + "=" + t
		outm[p] = t
	}
	return strings.Trim(out, "&"), outm
}



