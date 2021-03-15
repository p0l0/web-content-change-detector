package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var htmlBody = []byte(`<!DOCTYPE html>
	<html>
	<head>
	<link rel="stylesheet" href="styles.css">
	</head>
	<body>

	<h1>This is a heading</h1>
	<p>This is a paragraph.</p>

	</body>
	</html>`)

var htmlBodyNewSpaces = []byte(`<!DOCTYPE html>
	<html>
	<head>
	<link rel="stylesheet" href="styles.css">
	</head>
	<body>

	  <h1>This is a heading</h1>
	<p>This is a paragraph.</p>  

	</body>
	</html>`)

var htmlBodyNew = []byte(`<!DOCTYPE html>
	<html>
	<head>
	<link rel="stylesheet" href="styles.css">
	</head>
	<body>

	  <h1>This is a new heading</h1>
	<p>This is a new paragraph.</p>  

	</body>
	</html>`)

func TestGetContentSuccess(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write(htmlBody)
	}))
	defer func() { testServer.Close() }()

	response, err := getContent(testServer.URL)
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, htmlBody, response)
}

func TestGetContentRequestError(t *testing.T) {
	invalidURL := "http:// test.com"
	response, err := getContent(invalidURL)
	expectedError := "parse \"" + invalidURL + "\": invalid character \" \" in host name"
	assert.Equal(t, expectedError, err.Error())
	assert.Nil(t, response)
}


func TestGetContentURLError(t *testing.T) {
	invalidURL := "test.com"
	response, err := getContent(invalidURL)
	expectedError := "Error getting Response: Get \"" + invalidURL + "\": unsupported protocol scheme \"\""
	assert.Equal(t, expectedError, err.Error())
	assert.Nil(t, response)
}

func TestGetContentStatusError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(404)
		res.Write([]byte(""))
	}))
	defer func() { testServer.Close() }()

	response, err := getContent(testServer.URL)
	expectedError := "Incorrect HTTP Status Code: 404 Not Found"
	assert.Equal(t, expectedError, err.Error())
	assert.Nil(t, response)
}

func TestInitializeDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS responseData \\(url text, crawlTime text, response text\\);").WillReturnResult(sqlmock.NewResult(1, 1))

	err = initializeDB(db)
	require.NoError(t, err, "Expected no error")

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInitializeDBError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	err = initializeDB(db)
	require.Error(t, err, "Expected Error")
}

func TestInsertRecoredData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	scanUrl := "http://www.test.com"

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO responseData")
	mock.ExpectExec("INSERT INTO responseData").
		WithArgs(scanUrl, fmt.Sprintf("%s", htmlBody)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = insertRecoredData(db, scanUrl, htmlBody)
	require.NoError(t, err, "Expected no error")

	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInsertRecoredBeginError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	scanUrl := "http://www.test.com"

	// Test 'BEGIN' error
	err = insertRecoredData(db, scanUrl, htmlBody)
	assert.Equal(t, "all expectations were already fulfilled, call to database transaction Begin was not expected", err.Error())
}

func TestInsertRecoredPrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	scanUrl := "http://www.test.com"

	// Test 'PREPARE' error
	mock.ExpectBegin()
	err = insertRecoredData(db, scanUrl, htmlBody)
	assert.Equal(t, "all expectations were already fulfilled, call to Prepare 'INSERT INTO responseData(url, crawlTime, response) values(?, datetime('now'), ?)' query was not expected", err.Error())
}

func TestInsertRecoredInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	scanUrl := "http://www.test.com"

	// Test 'INSERT' error
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO responseData")
	mock.ExpectRollback()

	err = insertRecoredData(db, scanUrl, htmlBody)
	assert.Equal(t, "call to ExecQuery 'INSERT INTO responseData(url, crawlTime, response) values(?, datetime('now'), ?)' with args [{Name: Ordinal:1 Value:http://www.test.com} {Name: Ordinal:2 Value:<!DOCTYPE html>\n\t<html>\n\t<head>\n\t<link rel=\"stylesheet\" href=\"styles.css\">\n\t</head>\n\t<body>\n\n\t<h1>This is a heading</h1>\n\t<p>This is a paragraph.</p>\n\n\t</body>\n\t</html>}], was not expected, next expectation is: ExpectedRollback => expecting transaction Rollback", err.Error())

	// Make sure 'Rollback' was executed!
	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetLastEntries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	requestURL := "http://www.test.com"

	columns := []string{"url", "crawlTime", "response"}
	var expectedResult []dbRow
	expectedResult = append(expectedResult, dbRow{
		url: requestURL,
		crawlTime: "2019-01-10 14:02:10",
		response: fmt.Sprintf("%s", htmlBody),
	})
	expectedResult = append(expectedResult, dbRow{
		url: requestURL,
		crawlTime: "2019-01-10 14:06:10",
		response: fmt.Sprintf("%s - 1", htmlBody),
	})

	mock.ExpectQuery(`SELECT url, datetime\(crawlTime\) AS crawlTime, response FROM responseData WHERE url = \? ORDER BY crawlTime DESC LIMIT 2`).
		WithArgs(requestURL).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(expectedResult[0].url, expectedResult[0].crawlTime, expectedResult[0].response).
			AddRow(expectedResult[1].url, expectedResult[1].crawlTime, expectedResult[1].response))

	resultData, err := getLastEntries(db, requestURL)
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, expectedResult, resultData)
}

func TestGetDifferencesEqual(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf("%s", htmlBody))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, differences{}, diffs)
}

func TestGetDifferencesEqualError(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf("%s", htmlBody))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, differences{}, diffs)
}

func TestGetDifferencesNotEqual(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf("%s", htmlBodyNew))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t,
		differences{
			text:"--- Old\n+++ Current\n@@ -5,8 +5,8 @@\n </head>\n <body>\n \n-<h1>This is a heading</h1>\n-<p>This is a paragraph.</p>\n+<h1>This is a new heading</h1>\n+<p>This is a new paragraph.</p>\n \n </body>\n </html>\n",
			html:"<span>--- Old<br />+++ Current<br />@@ -5,8 +5,8 @@<br /> &lt;/head&gt;<br /> &lt;body&gt;<br /> <br />-&lt;h1&gt;This is a heading&lt;/h1&gt;<br />-&lt;p&gt;This is a paragraph.&lt;/p&gt;<br />+&lt;h1&gt;This is a new heading&lt;/h1&gt;<br />+&lt;p&gt;This is a new paragraph.&lt;/p&gt;<br /> <br /> &lt;/body&gt;<br /> &lt;/html&gt;<br /></span>"},
		diffs)
}

func TestGetDifferencesWithSpacesEqual(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf(" %s ", htmlBodyNewSpaces))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t,
		differences{
			text:"",
			html:""},
		diffs)
}

func startFakeSMTPServer(t *testing.T, fromEmail string, toEmail string, parseURL string) {
	listener, err := net.Listen("tcp", ":587")
	assert.NoError(t, err, "Expected no error on creating Listener")

	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err, "Expected no error on starting Listener")

		go func(conn net.Conn) {
			conn.Write([]byte("220 testdomain.com ESMTP\n"))
			defer conn.Close()

			for {
				// Make a buffer to hold incoming data.
				buf := make([]byte, 1024)

				// Read the incoming connection into the buffer.
				_, err := conn.Read(buf)
				if err != nil {

					if err == io.EOF {
						fmt.Println("Connection Closed")
						break
					}

					break
				}
				clientString := strings.Trim(string(buf), "\x00")

				// Get SMTP command as single string
				command := strings.Trim(strings.Split(clientString, " ")[0], "\r\n")

				switch command {
				case "EHLO":
					fakeSMTPEHLOCommand(t, conn, clientString)
				case "STARTTLS":
					conn = fakeSMTPSTARTTLSCommand(t, conn, clientString)
				case "MAIL":
					fakeSMTPMAILCommand(t, conn, clientString, fromEmail)
				case "RCPT":
					fakeSMTPRCPTCommand(t, conn, clientString, toEmail)
				case "DATA":
					fakeSMTPDATACommand(t, conn, clientString)
				case "QUIT":
					fakeSMTPQUITCommand(t, conn, clientString)
				default:
					assert.Equal(t, "", command, "Unknown command received")
				}
			}
		}(conn)

	}()
}

func fakeSMTPEHLOCommand(t *testing.T, conn net.Conn, clientString string) {
	assert.Equal(t, "EHLO localhost\r\n", clientString)
	conn.Write([]byte("250-testdomain.com\n250-PIPELINING\n250-SIZE 209715200\n250-VRFY\n250-ETRN\n250-STARTTLS\n250-ENHANCEDSTATUSCODES\n250-8BITMIME\n250 DSN\n"))
}

func fakeSMTPSTARTTLSCommand(t *testing.T, conn net.Conn, clientString string) net.Conn {
	assert.Equal(t, "STARTTLS\r\n", clientString)
	conn.Write([]byte("220 2.0.0 Ready to start TLS\n"))
	var tlsConn *tls.Conn
	cert, err := tls.LoadX509KeyPair("testdata/testdomain.com/cert.pem", "testdata/testdomain.com/key.pem")
	if err != nil {
		log.Fatal(err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	tlsConn = tls.Server(conn, cfg)
	return net.Conn(tlsConn)
}

func fakeSMTPMAILCommand(t *testing.T, conn net.Conn, clientString string, fromEmail string) {
	assert.Equal(t, "MAIL FROM:<" + fromEmail + "> BODY=8BITMIME\r\n", clientString)
	conn.Write([]byte("250 Ok\n"))
}

func fakeSMTPRCPTCommand(t *testing.T, conn net.Conn, clientString string, toEmail string) {
	assert.Equal(t, "RCPT TO:<" + toEmail + ">\r\n", clientString)
	conn.Write([]byte("250 Ok\n"))
}

func fakeSMTPDATACommand(t *testing.T, conn net.Conn, clientString string) {
	assert.Equal(t, "DATA\r\n", clientString)
	conn.Write([]byte("354 send the mail data, end with .\n"))

	readingData := true
	var emailContent string

	for readingData == true {
		// Make a buffer to hold incoming data.
		buf := make([]byte, 1024)

		// Read the incoming connection into the buffer.
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
		}
		clientString = strings.Trim(string(buf), "\x00")

		splitted := strings.SplitAfter(clientString, "\r\n")

		emailContent = emailContent + clientString
		if strings.Trim(splitted[len(splitted)-2], "\r\n") == "." {
			assert.Regexp(t, `Mime-Version: 1\.0\r\nDate: [A-Za-z]{3}, [0-9]{1,2} [A-Za-z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} \+[0-9]{4}\r\n((From: from@test\.com\r\n)|(To: to@test\.com\r\n)|(Subject: Change detected on URL: https://www\.test\.com\r\n)){3}Content-Type: multipart/alternative;\r\n boundary=[0-9a-z]+\r\n\r\n--[0-9a-z]+\r\nContent-Transfer-Encoding: quoted-printable\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n(&lt;|<)span(&gt;|>)--- Old(&lt;|<)br /(&gt;|>)\+\+\+ Current(&lt;|<)br /(&gt;|>)@@ -5,8 \+5,8 @@(&lt;|<)br /(&gt;|>) (&lt;|<)/head(&gt;|>)(&lt;|<)br =\r\n/(&gt;|>) (&lt;|<)body(&gt;|>)(&lt;|<)br /(&gt;|>) (&lt;|<)br /(&gt;|>)-(&lt;|<)h1(&gt;|>)This is a heading(&lt;|<)/h1(&gt;|>)(&lt;|<)br /(&gt;|>)-(&|<)=\r\n(lt;|)p(&gt;|>)This is a paragraph\.(&lt;|<)/p(&gt;|>)(&lt;|<)br /(&gt;|>)\+(&lt;|<)h1(&gt;|>)This is a new headin=\r\ng(&lt;|<)/h1(&gt;|>)(&lt;|<)br /(&gt;|>)\+(&lt;|<)p(&gt;|>)This is a new paragraph\.(&lt;|<)/p(&gt;|>)(&lt;|<)br /(&gt;|>) (&lt;|<)br /(&gt;|>)=\r\n (&lt;|<)/body(&gt;|>)(&lt;|<)br /(&gt;|>) (&lt;|<)/html(&gt;|>)(&lt;|<)br /(&gt;|>)(&lt;|<)/span(&gt;|>)\r\n--[0-9a-z]+\r\nContent-Transfer-Encoding: quoted-printable\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n--- Old\r\n\+\+\+ Current\r\n@@ -5,8 \+5,8 @@\r\n </head>\r\n <body>\r\n=20\r\n-<h1>This is a heading</h1>\r\n-<p>This is a paragraph\.</p>\r\n\+<h1>This is a new heading</h1>\r\n\+<p>This is a new paragraph\.</p>\r\n=20\r\n </body>\r\n </html>\r\n\r\n--[0-9a-z]+--\r\n.\r\n`, emailContent)
			conn.Write([]byte("250 Ok\n"))
			readingData = false
		}
	}
}

func fakeSMTPQUITCommand(t *testing.T, conn net.Conn, clientString string) {
	assert.Equal(t, "QUIT\r\n", clientString)
	conn.Write([]byte("221 Bye\n"))
	conn.Close()
}

func TestSendEmail(t *testing.T) {
	diff := differences{
		text: "--- Old\n+++ Current\n@@ -5,8 +5,8 @@\n </head>\n <body>\n \n-<h1>This is a heading</h1>\n-<p>This is a paragraph.</p>\n+<h1>This is a new heading</h1>\n+<p>This is a new paragraph.</p>\n \n </body>\n </html>\n",
		html: "<span>--- Old<br />+++ Current<br />@@ -5,8 +5,8 @@<br /> &lt;/head&gt;<br /> &lt;body&gt;<br /> <br />-&lt;h1&gt;This is a heading&lt;/h1&gt;<br />-&lt;p&gt;This is a paragraph.&lt;/p&gt;<br />+&lt;h1&gt;This is a new heading&lt;/h1&gt;<br />+&lt;p&gt;This is a new paragraph.&lt;/p&gt;<br /> <br /> &lt;/body&gt;<br /> &lt;/html&gt;<br /></span>",
	}

	// Fake SMTP Server
	startFakeSMTPServer(t, "from@test.com", "to@test.com", "https://www.test.com")

	CA_Pool := x509.NewCertPool()
	serverCert, err := ioutil.ReadFile("./testdata/cert.pem")
	if err != nil {
		log.Fatal("Could not load root ca!")
	}
	CA_Pool.AppendCertsFromPEM(serverCert)
	tlsConfig := tls.Config{RootCAs: CA_Pool, ServerName: "testdomain.com"}

	sendEmail(diff, "from@test.com", "to@test.com", "https://www.test.com", &tlsConfig)
}
