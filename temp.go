package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"crypto/md5"
	"github.com/go-ini/ini"
)


var OtherOptions map[string]string

var Basedir string

var Host string

func main() {
	//设置起始链接
	//firstUrl := "http://www.gzweixin168.com"
	getHtml( Host )
}
func init(){
	cfg, err := ini.Load("config.ini")
	if err != nil{
		panic("fatal err, can't load ini file")
	}
	baseSection := cfg.Section("base")
	if !baseSection.HasKey("basedir") || !baseSection.HasKey("host"){
		panic("basedir or host not set")
	}
	Basedir = baseSection.Key("basedir").String()
	Host = baseSection.Key("host").String()
	//Basedir = "/Users/kazuma/Desktop/Html"
	//Host = "http://www.gzweixin168.com"
	OtherOptions = cfg.Section("others").KeysHash()
}

func getHtml(htmlUrl string) {

	//flag.Parse()
	//inputUrl := flag.Arg(0)

	//parse domain
	u, err := url.Parse(htmlUrl)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(u.Scheme)
	//fmt.Println(u.Host)
	//fmt.Println(u.Port())

	rootUrl := u.Scheme + "://" + u.Host
	if u.Port() != "" {
		rootUrl += ":" + u.Port()
	}
	resp, err := http.Get(htmlUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("%s status code error: %d %s \n %s \n", htmlUrl, resp.StatusCode, resp.Status , htmlUrl )
	}
	//body, err := ioutil.ReadAll(resp.Body)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("start to load included files")

	// Find the review items
	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		fmt.Println("get link")
		// For each item found, get the band and title
		v, ok := s.Attr("href")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file

			downloadCss(v, rootUrl, Basedir)
		}

	})

	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		fmt.Println("get script")
		v, ok := s.Attr("src")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file
			downloadSrc(v, rootUrl, Basedir)
		}

	})

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		fmt.Println("get img")
		// For each item found, get the band and title
		v, ok := s.Attr("src")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file
			downloadSrc(v, rootUrl, Basedir)
		}

	})


	//page := filepath.Base(u.Path)
	path := u.Path
	query := u.RawQuery
	query = strings.Replace( query ,"&" , "_" , -1 )
	query = strings.Replace( query ,"=" , "_" , -1 )
	//query = strings.Replace( query ,"?" , "_" , -1 )

	if  path == "/" || path == ""{
		path = "index"
	}

	path = strings.Replace(path,"/./","_", -1)
	path = strings.Replace(path,"/","_", -1)
	path = strings.Replace(path,".","_", -1)
	path = path + query +".html"
	oFile, err := os.Create(Basedir + "/" + path)
	if err != nil {
		log.Fatal(err)
	}
	html, err := doc.Html()
	if err == nil {
		transferredHtml := htmlLinkReplace(html)
		for key, val := range OtherOptions{
			transferredHtml = strings.Replace(transferredHtml, key, val,-1)
		}
		oFile.WriteString(transferredHtml)
		//oFile.WriteString(transferredBody)

		//提取页面的图片
		re, err := regexp.Compile("url\\((.+?)\\)")
		if err != nil {
			log.Fatal(err)
		}

		allMatch := re.FindAllStringSubmatch(html, -1)
		for _, v := range allMatch {
			downloadSrc(v[1], rootUrl, Basedir)
		}

	}
	//创建锁文件
	en := md5.New()
	io.WriteString( en , u.RawQuery )
	md5str2 := fmt.Sprintf("%x", en.Sum(nil))
	fmt.Println( md5str2 )
	createLock( md5str2 )
	fmt.Println("Get Body Success")

	fmt.Println("Get Next Link Start")
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		v, ok := s.Attr("href")
		if ok {
			if strings.HasPrefix(v,"http") && !strings.HasPrefix(v,Host){
				return
			}
			//判断这个地址是否已经存在如果存在则跳过 ， 如果不存在则去抓取页面 ， 直到所有的页面获取完比 递归调用
			if !strings.HasPrefix( v, "http") {
				if !strings.HasPrefix(v, "/") {
					v = "/" + v
				}
				v = rootUrl + v
			}

			uv , err := url.Parse( v )
			if err != nil {
				log.Fatal( err )
			}

			fmt.Println( uv.RawQuery )
			en := md5.New()
			io.WriteString( en , uv.RawQuery )
			md5str2 := fmt.Sprintf("%x", en.Sum(nil))
			fmt.Println( md5str2 )
			//如果锁不存在则获取数据
			if !lockExists( md5str2 ) {
				getHtml( v )
			}
		}
	})



}

/**
 * 下载页面的css
 */
func downloadCss(cssUrl string, rootUrl string, baseDir string) bool {
	fmt.Println("get css")
	fullcssUrl := cssUrl
	if !strings.HasPrefix(fullcssUrl, "http") {
		if !strings.HasPrefix(fullcssUrl, "/") {
			fullcssUrl = "/" + fullcssUrl
		}
		fullcssUrl = rootUrl + fullcssUrl
	}
	u, err := url.Parse(fullcssUrl)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(u.Path)
	fullPath := baseDir + u.Path
	if isExists( fullPath ) {
		return true
	}
	res, err := http.Get(fullcssUrl)

	if err != nil {
		log.Fatal(err)

	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("%s status code error: %d %s \n", fullcssUrl , res.StatusCode, res.Status)
		return false
	}

	//创建文件

	dirPath := filepath.Dir(fullPath)
	//检测文件是否存在
	if !isExists(dirPath) {
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			//产生错误
			fmt.Println(err)
			log.Fatal(err)
		}

	}

	oFile, err := os.Create(fullPath)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	oFile.Write(body)

	content := string(body)
	re, err := regexp.Compile("url\\((.+?)\\)")
	if err != nil {
		log.Fatal(err)
	}

	allMatch := re.FindAllStringSubmatch(content, -1)
	for _, v := range allMatch {
		if strings.HasPrefix(v[1], "/") {
			downloadSrc(v[1], rootUrl, Basedir)
		} else {
			dir := filepath.Dir(cssUrl)
			downloadSrc(dir+"/"+v[1], rootUrl, Basedir)
		}

	}

	return true
}

func downloadSrc(js string, rootUrl string, baseDir string) bool {
	//check js has scheme
	fmt.Println("get js")
	if !strings.HasPrefix(js, "http") {
		if !strings.HasPrefix(js, "/") {
			js = "/" + js
		}
		js = rootUrl + js
	}
	u, err := url.Parse(js)

	if err != nil {
		log.Fatal(err)
	}
	fullPath := baseDir + u.Path
	if isExists( fullPath ) {
		return true
	}
	res, err := http.Get(js)

	if err != nil {
		log.Fatal(err)

	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("status code error: %d %s \n %s \n", res.StatusCode, res.Status , js )
		return false
	}

	//创建文件
	dirPath := filepath.Dir(fullPath)
	//检测文件是否存在
	if !isExists(dirPath) {
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			//产生错误
			fmt.Println(err)
			log.Fatal(err)
		}

	}

	oFile, err := os.Create(fullPath)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	oFile.Write(body)

	return true
}

/**
 * 文件是否存在
 */
func isExists(path string) bool {
	path = strings.Replace( path , "//" , "/" , 0 )
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true

}

func lockExists( v string ) bool {
	fmt.Println( v )
	fullPath := Basedir + "/lock/" + v
	//检测文件是否存在
	if !isExists(fullPath) {
		return false
	}
	return true
}

func createLock( v string ) bool {
	fullPath := Basedir + "/lock/" + v
	fmt.Println( fullPath )
	dirPath := filepath.Dir(fullPath)
	//检测文件是否存在
	if !isExists(dirPath) {
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			//产生错误
			fmt.Println(err)
			log.Fatal(err)
		}
	}
	f , err := os.Create(fullPath)
	fmt.Println( "lock file : " + fullPath )
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString( "ok" )
	f.Close()
	return true
}

func htmlLinkReplace(htmlBody string,) (string) {
	//替换规则:先将所有的
	re := regexp.MustCompile(`(?s)href="([\S]+?)"`)
	transferredBody := re.ReplaceAllStringFunc(htmlBody, func(s string) string {
		s = re.ReplaceAllString(s,"$1")
		s = strings.Replace(s,"./","_", -1)
		s = strings.Replace(s,"/","_", -1)
		s = strings.Replace(s,".","_", -1)
		s = strings.Replace(s, "&amp;", "_",-1)
		s = strings.Replace(s, "=", "_",-1)
		s = strings.Replace(s, "?", "",-1)
		fmt.Println("transffered link is", "href=\"index"+s+".html\"")
		if s == "_"{
			s = "index"
		}
		return "href=\""+s+".html\""
	})
	return transferredBody
}

