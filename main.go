package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pmezard/go-difflib/difflib"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"html"
	"time"
)

type dbRow struct {
	url string
	crawlTime string
	response string
}

type differences struct {
	text string
	html string
}

func getContent(scanUrl string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	request, err := http.NewRequest("GET", scanUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("Accept-Language", "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7,es;q=0.6")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Error getting Response: %s", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Incorrect HTTP Status Code: %s", response.Status)
	}


	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read Response: %s", err)
	}

	return body, nil
}

func initializeDB(db *sql.DB) error {
	sqlStmt := `
		CREATE TABLE IF NOT EXISTS responseData (url text, crawlTime text, response text);
	`

	_, err := db.Exec(sqlStmt)
	if err != nil {
		return err
	}

	return nil
}

func insertRecoredData(db *sql.DB, scanUrl string, response []byte) (error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO responseData(url, crawlTime, response) values(?, datetime('now'), ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(scanUrl, fmt.Sprintf("%s", response))
	if err != nil {
		return err
	}
	tx.Commit()

	return nil
}

func getLastEntries(db *sql.DB) ([]dbRow, error) {
	rows, err := db.Query("SELECT url, datetime(crawlTime) AS crawlTime, response FROM responseData ORDER BY crawlTime DESC LIMIT 2")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultData []dbRow
	for rows.Next() {
		var rowResult = dbRow{}
		err = rows.Scan(&rowResult.url, &rowResult.crawlTime, &rowResult.response)
		if err != nil {
			return nil, err
		}

		resultData = append(resultData, rowResult)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return resultData, nil
}

func getDifferences(response1 string, response2 string) (differences, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(response1),
		B:        difflib.SplitLines(response2),
		FromFile: "Old",
		ToFile:   "Current",
		Context:  3,
		Eol:      "\n",
	}
	var result differences
	var err error

	result.text, err = difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return result, err
	}
	if result.text != "" {
		result.html = "<span>" + strings.Replace(html.EscapeString(result.text), "\n", "<br />", -1) + "</span>"
	}

	return result, nil
}

func sendEmail(diffs differences, fromEmail string, toEmail string, url string, smtpTLSHost string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", fromEmail)
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", "Change detected on URL: " + url)
	message.SetBody("text/html", diffs.html)
	message.AddAlternative("text/plain", diffs.text)

	mail := gomail.Dialer{Host: "localhost", Port: 587, TLSConfig: &tls.Config{ServerName: smtpTLSHost}}
	err := mail.DialAndSend(message)
	if err != nil {
		return fmt.Errorf("Unable to send Email: %s", err)
	}

	return nil
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

	if *fromEmail == "" {
		log.Fatal("Please specify a Sender email")
	}

	if *smtpTLSHost == "" {
		log.Fatal("Please specify the TLS SMTP Domain")
	}

	response, err := getContent(*scanUrl)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "file:data.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = initializeDB(db)
	if err != nil {
		log.Fatal(err)
	}

	err = insertRecoredData(db, *scanUrl, response)
	if err != nil {
		log.Fatal(err)
	}

	resultData, err := getLastEntries(db)

	if len(resultData) < 2 {
		log.Println("Not enough Data crawled for comparing")
		return
	} else if len(resultData) != 2 {
		log.Fatal("We got to much entries from Database: ", len(resultData))
	}

	diffs, err := getDifferences(resultData[0].response, resultData[1].response)
	if err != nil {
		log.Fatal(err)
	}

	if (differences{}) != diffs {
		err = sendEmail(diffs, *fromEmail, *toEmail, resultData[0].url, *smtpTLSHost)
		if err != nil {
			log.Fatal(err)
		}
	}
}
