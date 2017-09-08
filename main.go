package main

import (
    "fmt"
    "log"
    "github.com/PuerkitoBio/goquery"
    "sync"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "time"
)

type News struct {
    ID bson.ObjectId `bson:"_id,omitempty"`
    Provider string
    Url string
    Image string
    Title string
    CreatedAt time.Time
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
    // Parallelize(ettoday, appledaily)
    ettoday()
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

 func appledaily() {
    doc, err := goquery.NewDocument("http://www.appledaily.com.tw/realtimenews/section/new/")

    if err != nil {
      log.Fatal(err)
    }

    doc.Find(".item").Each(func(i int, s *goquery.Selection) {
        // date := s.Find("span").Text()
        title, _ := s.Find("img").Attr("alt")
        url, _ := s.Find("a").Attr("href")
        fmt.Printf("Review %d: %s - %s - %s\n", i+1, url, title)
    })
 }