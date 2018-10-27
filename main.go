package main

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)


var OtherOptions map[string]string

var Basedir string
var PathSeparator string
var Url string

func main() {
	//检查目标目录是否存在
	if os.IsPathSeparator('/') {
		PathSeparator = "/"
	} else {
		PathSeparator = "\\"
	}

	createDir( Basedir )


	urls := strings.Split( Url , ",")
	if len( urls ) > 0 {
		for _ , v := range urls {

			//分析路径
			u , err := url.Parse( v )
			if err != nil {
				fmt.Println( err )
				panic(fmt.Sprintf("url %s is not correct" , v ) )
			}

			childDir := Basedir + PathSeparator + u.Hostname()
			if u.Port() != "80" && u.Port() != ""  {
				childDir = childDir + "_" + u.Port()
			}
			childDir += PathSeparator
			createDir( childDir )
			//抓取页面
			getHtml( v , childDir )
		}
	} else {
		panic( "url is empty")
	}

}
func init(){
	cfg, err := ini.Load("config.ini")
	if err != nil{
		panic("fatal err, can't load ini file")
	}
	baseSection := cfg.Section("base")
	if !baseSection.HasKey("basedir"){
		panic("basedir not set")
	}
	if !baseSection.HasKey("url"){
		panic("url not set")
	}
	Basedir = baseSection.Key("basedir").String()
	Url = baseSection.Key("url").String()
	OtherOptions = cfg.Section("others").KeysHash()
}


//创建目录
func createDir ( dir string ) (bool , error) {
	dirPath := filepath.Dir( dir )
	if !isExists( dirPath ) {
		err := os.MkdirAll( dirPath , os.ModePerm)
		if err != nil {
			return false , err
		}
	}
	return true , nil
}

func getHtml(htmlUrl string , currentDir string ) {

	//如果解析失败
	u, err := url.Parse(htmlUrl)
	if err != nil {
		panic("url incorrect")
	}

	if u.Host == "" {
		panic( fmt.Sprintf("Host not find in %s" , htmlUrl ) )
	}

	rootUrl := u.Scheme + "://" + u.Host
	if u.Port() != "" {
		rootUrl += ":" + u.Port()
	}
	resp, err := http.Get(htmlUrl)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic( fmt.Sprintf( "%s status code error: %d %s \n %s \n", htmlUrl, resp.StatusCode, resp.Status , htmlUrl ) )
	}

	//body, err := ioutil.ReadAll(resp.Body)
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic( err )
	}

	fmt.Println("start to load included files")
	// Find the review items

	fmt.Println("get link")
	doc.Find("link").Each(func(i int, s *goquery.Selection) {

		v, ok := s.Attr("href")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file


			downloadCss( v, rootUrl , currentDir )
		}

	})

	fmt.Println("get script")
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		v, ok := s.Attr("src")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file
			downloadSrc(v, rootUrl, currentDir)
		}

	})

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		fmt.Println("get img")
		// For each item found, get the band and title
		v, ok := s.Attr("src")
		if ok {
			fmt.Printf("Review %d: %s \n", i, v)
			//download css and write to file
			downloadSrc(v, rootUrl, currentDir )
		}

	})

	path := u.Path
	query := u.RawQuery
	query = strings.Replace( query ,"&" , "_" , -1 )
	query = strings.Replace( query ,"=" , "_" , -1 )

	if  path == "/" || path == ""{
		path = "index"
	}

	createDir( currentDir + path )

	path = path + query +".html"
	oFile, err := os.Create( currentDir + PathSeparator + path)
	if err != nil {
		log.Fatal(err)
	}
	html, err := doc.Html()
	if err == nil {
		oFile.WriteString( html )

		//提取页面的图片
		re, err := regexp.Compile("url\\((.+?)\\)")
		if err != nil {
			log.Fatal(err)
		}

		allMatch := re.FindAllStringSubmatch(html, -1)
		for _, v := range allMatch {
			downloadSrc(v[1], rootUrl, currentDir )
		}

	}
	fmt.Println("Get Body Success")
}


/**
 * 拼合路径
 */

 func checkUrl( resourceUrl string , rootUrl string ) (string , error ) {

	 u , err := url.Parse( rootUrl )
	 if err != nil {
		 return "" , err
	 }

	 //baseUrl := u.Scheme + "://" + u.Host

	 //如果是自适应型scheme
	 if strings.HasPrefix( resourceUrl , "://" ) {
		 resourceUrl = u.Scheme + resourceUrl
	 }


	 //如果是base64位编码的资源也不需要处理 url
	 if strings.HasPrefix( resourceUrl , "data:image" ) {
	 	return "" , errors.New("current resource is base64encode , there is no need to download")
	 }


	 //如果是不带Http的请求 需要拼接成完整的url地址
	 if !strings.HasPrefix(resourceUrl, "http") {

		 ru , err := url.Parse( resourceUrl )

		 if err != nil {
			 return "" , err
		 }

		 nResourceUrl := u.ResolveReference( ru )

		 resourceUrl = nResourceUrl.String()

	 }
	 return resourceUrl , nil
 }

/**
 * 下载页面的css
 */
func downloadCss(cssUrl string, rootUrl string, baseDir string) bool {

	rootUrlParse , err := url.Parse( rootUrl )
	if err != nil {
		panic( err )
	}

	fullcssUrl , err := checkUrl( cssUrl , rootUrl )

	if err != nil {
		fmt.Println(err)
		return false
	}

	baseUrl := rootUrlParse.Scheme + "://" + rootUrlParse.Host


	//检查是不是本站资源 ，如果不是则不需要下载
	if !strings.HasPrefix( fullcssUrl , baseUrl ) {
		fmt.Sprintf( "current resource is %s , we skip download" , fullcssUrl )
		return true
	}

	u, err := url.Parse(fullcssUrl)

	if err != nil {
		panic(err)
	}

	fullPath := baseDir + u.Path
	if isExists( fullPath ) {
		return true
	}
	res, err := http.Get(fullcssUrl)

	if err != nil {
		fmt.Println(err)
		return false

	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("%s status code error: %d %s \n", fullcssUrl , res.StatusCode, res.Status)
		return false
	}

	//创建文件
	ok , _ := createDir( fullPath )
	if ok {
		oFile, err := os.Create(fullPath)
		if err != nil {
			log.Fatal(err)
		}
		body, err := ioutil.ReadAll(res.Body)
		oFile.Write(body)

		//download css里面的图片资源
		content := string(body)
		re, err := regexp.Compile("url\\((.+?)\\)")
		if err != nil {
			log.Fatal(err)
		}

		allMatch := re.FindAllStringSubmatch(content, -1)
		for _, v := range allMatch {
			//检查资源是否是路径 ，如果不是则不下载
			downloadSrc(v[1], fullcssUrl , Basedir)
		}

		return true
	}
	return true
}

func downloadSrc(js string, rootUrl string, baseDir string) bool {
	//check js has scheme
	js , err := checkUrl( js , rootUrl )

	if err != nil {
		fmt.Println(err)
		return false
	}

	//检查是不是本站资源 ，如果不是则不需要下载
	if !strings.HasPrefix( js , rootUrl ) {
		fmt.Sprintf( "current resource is %s , we skip download" , js )
		return true
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