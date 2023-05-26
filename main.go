package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ExampleScrape() []Item {
	var list []Item
	// Request the HTML page.
	res, err := http.Get("http://www.chncpa.org/yspj_260/zmylh_262/")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find(".yspj-title-a").Each(func(i int, s *goquery.Selection) {
		title, titleExist := s.Attr("title")
		href, hrefExist := s.Attr("href")
		if titleExist && hrefExist {
			list = append(list, Item{title, href})
		}
	})
	return list
}

type Item struct {
	title string
	url   string
}

type Task struct {
	title  string
	url    string
	text   string
	open   bool
	notify bool
}

func main() {
	fmt.Println("start to fetch concert list")
	list := ExampleScrape()
	//list := []*Item{{"duanjin", "http://ticket.chncpa.org/product-1096150.html"}}
	for i, item := range list {
		fmt.Printf("%d: %s \n", i, item.title)
	}

	fmt.Println("enter select concert index")
	var selectConcerts string
	fmt.Scanln(&selectConcerts)

	concerts := strings.Split(selectConcerts, ",")

	var taskList []*Task
	for _, item := range concerts {
		var task Task
		index, err := strconv.Atoi(item)
		if err != nil {
			continue
		}
		task.url = list[index].url
		task.title = list[index].title
		task.text = ""
		task.open = false
		task.notify = false
		taskList = append(taskList, &task)
	}

	Ticker(&taskList)
}

func Ticker(taskList *[]*Task) {

	fmt.Println("请输入您想要循环的时间间隔（秒）：\r\n")
	var interval int
	fmt.Scanln(&interval)

	fmt.Println("任务已启动，请按下 Ctrl+C 停止任务")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	var counter = 1 * interval

	progressCh := make(chan int)
	go func(taskList *[]*Task) {
		for {
			select {
			case progress := <-progressCh:
				for _, task := range *taskList {
					singleTask(task)
				}
				fmt.Println(progress)
			}
		}
	}(taskList)

	for {
		select {
		case <-ticker.C:
			progressCh <- counter
			counter = counter + interval
		}
	}
}

func singleTask(task *Task) {
	if task.notify {
		return
	}
	res, err := http.Get(task.url)
	fmt.Println(task.text)
	if err != nil {
		log.Fatal(err)
		fmt.Printf("%s fetch fail", task.title)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		fmt.Printf("%s fetch fail", task.title)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
		fmt.Printf("%s parse fail", task.title)
	}
	doc.Find(".tick-xin-top h1 span").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		task.text = text
		fmt.Println(task)
		if !strings.Contains(text, "即将开票") {
			// notification
			if task.notify {
				return
			}

			data := make(map[string]interface{})
			data["appToken"] = "AT_XEjhyWYQ4uhcecm0mStXRNWTNO9xFsSI"
			data["content"] = task.title + "已经开票 !!!"
			data["contentType"] = 1
			data["uids"] = [...]string{"UID_yxfTSW0zm1MbLOTuRYRQyQfk1ARx"}
			data["url"] = task.url
			data["verifyPay"] = false
			data["summary"] = "有音乐会开票，点击查看 !!!"
			bytesData, _ := json.Marshal(data)
			http.Post("https://wxpusher.zjiecode.com/api/send/message", "application/json", bytes.NewReader(bytesData))

			task.notify = true
			task.open = true
		}
	})
}
