package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type dbRow struct {
	url string
	crawlTime string
	response string
}

func main() {
	scanUrl := flag.String("url", "", "URL To Scan")
	toEmail := flag.String("to", "", "Email to send report to")
	fromEmail := flag.String("from", "", "Email to send report from")
	smtpTLSHost := flag.String("tlsHost", "", "Host to match TLS")

	flag.Parse()

	if *scanUrl == "" {
		log.Fatal("Please specify an URL to scan")
	}

	if *toEmail == "" {
		log.Fatal("Please specify a Report email")
	}

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	request, err := http.NewRequest("GET", *scanUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("Accept-Language", "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7,es;q=0.6")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36")

	response, err := client.Do(request)
	if err != nil {
		log.Fatal("Error getting Response", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Unable to read Response", err)
	}
	//fmt.Printf("%s\n", body)

	db, err := sql.Open("sqlite3", "file:data.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS responseData (url text, crawlTime text, response text);
	`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO responseData(url, crawlTime, response) values(?, datetime('now'), ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(*scanUrl, fmt.Sprintf("%s", body))
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()

	rows, err := db.Query("SELECT url, datetime(crawlTime), response FROM responseData ORDER BY crawlTime DESC LIMIT 2")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var resultData []dbRow
	for rows.Next() {
		var rowResult = dbRow{}
		err = rows.Scan(&rowResult.url, &rowResult.crawlTime, &rowResult.response)
		if err != nil {
			log.Fatal(err)
		}

		resultData = append(resultData, rowResult)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	if len(resultData) < 2 {
		log.Println("Not enough Data crawled for comparing")
		return
	} else if len(resultData) != 2 {
		log.Fatal("We got to much entries from Datbase")
	}

	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(resultData[0].response, resultData[1].response, false)

	// Maybe there is a better way to do this...
	changesDetected := false
	//fmt.Println(len(diffs))
	for i:=0; i < len(diffs); i++ {
		//fmt.Println(diffs[i].Type)
		if diffs[i].Type != diffmatchpatch.DiffEqual {
			changesDetected = true
			break
		}
	}
	//fmt.Println(changesDetected)

	//fmt.Println(dmp.DiffPrettyHtml(diffs))

	if changesDetected {
		message := gomail.NewMessage()
		message.SetHeader("From", *fromEmail)
		message.SetHeader("To", *toEmail)
		message.SetHeader("Subject", "Change detected on URL: " + resultData[0].url)
		message.SetBody("text/plain", dmp.DiffPrettyText(diffs))
		message.SetBody("text/html", dmp.DiffPrettyHtml(diffs))

		//err = message.Send()
		mail := gomail.Dialer{Host: "localhost", Port: 587, TLSConfig: &tls.Config{ServerName: *smtpTLSHost}}
		err = mail.DialAndSend(message)
		if err != nil {
			log.Fatal("Unable to send Email: ", err)
		}
	}
}
