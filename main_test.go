package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"net/http"
	"net/http/httptest"
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

func TestGetContentError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(404)
		res.Write([]byte(""))
	}))
	defer func() { testServer.Close() }()

	response, err := getContent(testServer.URL)
	expectedError := "Incorrect HTTP Status Code: 404 Not Found"
	assert.Equal(t, err.Error(), expectedError)
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

func TestGetLastEntries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"url", "crawlTime", "response"}
	var expectedResult []dbRow
	expectedResult = append(expectedResult, dbRow{
		url: "http://www.test.com",
		crawlTime: "2019-01-10 14:02:10",
		response: fmt.Sprintf("%s", htmlBody),
	})
	expectedResult = append(expectedResult, dbRow{
		url: "http://www.test.com",
		crawlTime: "2019-01-10 14:06:10",
		response: fmt.Sprintf("%s - 1", htmlBody),
	})

	mock.ExpectQuery("SELECT url, datetime\\(crawlTime\\) AS crawlTime, response FROM responseData ORDER BY crawlTime DESC LIMIT 2").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(expectedResult[0].url, expectedResult[0].crawlTime, expectedResult[0].response).
			AddRow(expectedResult[1].url, expectedResult[1].crawlTime, expectedResult[1].response))

	resultData, err := getLastEntries(db)
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, expectedResult, resultData)
}

func TestGetDifferencesEqual(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf("%s", htmlBody))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, differences{}, diffs)
}

func TestGetDifferencesNotEqual(t *testing.T) {
	diffs, err := getDifferences(fmt.Sprintf("%s", htmlBody), fmt.Sprintf("%s - 1", htmlBody))
	require.NoError(t, err, "Expected no error")
	assert.Equal(t,
		differences{
			text:"--- Old\n+++ Current\n@@ -9,4 +9,4 @@\n \t<p>This is a paragraph.</p>\n \n \t</body>\n-\t</html>\n+\t</html> - 1\n",
			html:"<span>--- Old<br />+++ Current<br />@@ -9,4 +9,4 @@<br /> \t&lt;p&gt;This is a paragraph.&lt;/p&gt;<br /> <br /> \t&lt;/body&gt;<br />-\t&lt;/html&gt;<br />+\t&lt;/html&gt; - 1<br /></span>"},
		diffs)
}

func TestSendEmail(t *testing.T) {
	//@todo - Look how to mock SMTP!
}