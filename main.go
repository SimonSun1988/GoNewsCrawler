package main

import (
    "fmt"
    "log"
    "strings"
    "strconv"
    "github.com/PuerkitoBio/goquery"
    "sync"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "time"
    "github.com/robfig/cron"
    "net/url"
)

type News struct {
    ID bson.ObjectId `bson:"_id,omitempty"`
    Provider string
    Url string
    Image string
    Title string
    CreatedAt time.Time
}

func removeQueryString(inputUrl string) string {
    u, err := url.Parse(inputUrl)
    if err != nil {
        log.Fatal(err)
    }
    u.RawQuery = ""
    return u.String()
}

// TODO: GoLang Parallelize
func Parallelize(functions ...func()) {
    var waitGroup sync.WaitGroup
    waitGroup.Add(len(functions))

    defer waitGroup.Wait()

    for _, function := range functions {
        go func(copy func()) {
            defer waitGroup.Done()
            copy()
        }(function)
    }
}

func main() {

    cronjob := cron.New()
    spec := "0 */3 * * * *"
    cronjob.AddFunc(spec, func() {
        Parallelize(ettoday, appledaily, udn)
    })
    cronjob.Start()
    select {}

    // Parallelize(ettoday, appledaily)
    // ettoday()
    // appledaily()
}

// 東森新聞雲
func ettoday() {

    // 開啟 mongodb connection，並且在 function 結束後關閉
    session, err := mgo.Dial("mongodb://localhost:27017")
    defer session.Close()

    // 爬即時新聞頁面
    doc, err := goquery.NewDocument("http://www.ettoday.net/news/news-list.htm")
    if err != nil {
        log.Fatal(err)
    }

    // 連線到 all_news db 與 news collection
    c := session.DB("all_news").C("news")

    err = c.EnsureIndexKey("url")
    err = c.EnsureIndexKey("-createdat")

    doc.Find(".part_list_2 h3").Each(func(i int, s *goquery.Selection) {

        // 找出所有即時新聞的連結
        url, _ := s.Find("a").Attr("href")

        // 爬出所有即時新聞的新聞內頁
        inner, err := goquery.NewDocument("http://www.ettoday.net" + url)
        if err != nil {
            log.Fatal(err)
        }

        // 取出標題，建立時間，圖片資訊
        title := s.Find("a").Text()
        image, _ := inner.Find(".story").Find("img").Attr("src")
        dateString := s.Find("span").Text()
        date, _ := time.Parse("2006/01/02 15:04", dateString)

        // 建立 news 物件
        news := News{"", "ettoday", url, image, title, date}

        // 用新聞連結找出資料
        result := News{}
        _ = c.Find(bson.M{"url": "http://www.ettoday.net" + news.Url}).One(&result)

        if result.Url != "" {
            fmt.Printf("已經存在的新聞資料: %s\n", "http://www.ettoday.net" + result.Url)
            fmt.Printf("時間: %s\n", result.CreatedAt)
            fmt.Println("------------------------\n")
        }  

        // 如果找不到這個新聞連結的資料，就幫他建立新的資料
        if result.Url == "" {
            fmt.Printf("建立新的新聞資訊: %s\n", "http://www.ettoday.net" + news.Url)
            fmt.Printf("時間: %s\n", date)
            fmt.Println("------------------------\n")
            _ = c.Insert(&News{
                Provider: news.Provider,
                Title: news.Title,
                Image: "https:" + news.Image,
                Url: "http://www.ettoday.net" + news.Url,
                CreatedAt: news.CreatedAt,
            })
        }
    })
}

// 蘋果日報
func appledaily() {

    // 開啟 mongodb connection，並且在 function 結束後關閉
    session, err := mgo.Dial("mongodb://localhost:27017")
    defer session.Close()
    // 連線到 all_news db 與 news collection
    newsCollection := session.DB("all_news").C("news")

    err = newsCollection.EnsureIndexKey("url")
    err = newsCollection.EnsureIndexKey("-createdat")

    doc, err := goquery.NewDocument("http://www.appledaily.com.tw/realtimenews/section/new/")

    if err != nil {
        log.Fatal(err)
    }

    doc.Find("li.rtddt").Each(func(i int, s *goquery.Selection) {
        url, _ := s.Find("a").Attr("href")

        innerNews, err := goquery.NewDocument("http://www.appledaily.com.tw" + url)
        if err != nil {
            log.Fatal(err)
        }

        title := innerNews.Find("#h1").Text()
        image, _ := innerNews.Find(".imgmid2 img").Attr("src")
        dateString := innerNews.Find(".gggs time").Text()
        date, _ := time.Parse("2006年01月02日15:04", dateString)

        // 用新聞連結找出資料
        result := News{}
        _ = newsCollection.Find(bson.M{"url": "http://www.appledaily.com.tw" + url}).One(&result)

        if result.Url != "" {
            fmt.Printf("已經存在的新聞資料: %s\n", result.Url)
            fmt.Printf("時間: %s\n", result.CreatedAt)
            fmt.Println("------------------------\n")
        }

        // 如果找不到這個新聞連結的資料，就幫他建立新的資料
        if result.Url == "" {
            fmt.Printf("建立新的新聞資訊: %s\n", "http://www.appledaily.com.tw" + url)
            fmt.Printf("時間: %s\n", date)
            fmt.Println("------------------------\n")
            _ = newsCollection.Insert(&News{
                Provider: "appledaily",
                Title: title,
                Image: image,
                Url: "http://www.appledaily.com.tw" + url,
                CreatedAt: date,
            })
        }
    })
}

// UDN
func udn() {
    // 開啟 mongodb connection，並且在 function 結束後關閉
    session, err := mgo.Dial("mongodb://localhost:27017")
    defer session.Close()
    if err != nil {
        log.Fatal(err)
    }

    // 連線到 all_news db 與 news collection
    newsCollection := session.DB("all_news").C("news")
    err = newsCollection.EnsureIndexKey("url")
    err = newsCollection.EnsureIndexKey("-createdat")

    doc, err := goquery.NewDocument("https://udn.com/news/breaknews/1/99")

    if err != nil {
        log.Fatal(err)
    }

    doc.Find("#breaknews_body dl dt").Each(func(i int, s *goquery.Selection) {

        // 找出 url 並且去掉 query string
        url, _ := s.Find("a").Attr("href")
        url = "https://udn.com" + removeQueryString(url)

        image, _ := s.Find("img").Attr("src")
        title := s.Find("h2").Text()

        // 拆解 date 並組成可用的時間格式
        dateString := s.Find(".info .dt").Text()
        thisYear := strconv.Itoa(time.Now().Year())
        newDateString := thisYear + "/" + strings.Replace(dateString, "-", "/", -1)
        date, _ := time.Parse("2006/01/02 15:04", newDateString)

        // fmt.Printf("title = %s\n", title)
        // fmt.Printf("url = %s\n", url)
        // fmt.Printf("image = %s\n", image)
        // fmt.Printf("date = %s\n", date)

        // 用新聞連結找出資料
        result := News{}
        _ = newsCollection.Find(bson.M{"url": url}).One(&result)

        if result.Url != "" {
            fmt.Printf("已經存在的新聞資料: %s\n", result.Url)
            fmt.Printf("時間: %s\n", result.CreatedAt)
            fmt.Println("------------------------\n")
        }

        // 如果找不到這個新聞連結的資料，就幫他建立新的資料
        if result.Url == "" {
            fmt.Printf("建立新的新聞資訊: %s\n", url)
            fmt.Printf("時間: %s\n", date)
            fmt.Println("------------------------\n")
            _ = newsCollection.Insert(&News{
                Provider: "udn",
                Title: title,
                Image: image,
                Url: url,
                CreatedAt: date,
            })
        }
    })
}