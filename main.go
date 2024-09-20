package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TheSgtPepper23/greenMigration/extras"
	"github.com/golang-jwt/jwt/v4"
)

// const baseurl = "https://bluefive.xyz/api/"
const baseurl = "http://localhost:5555/"
const secret = "b2B12@X9w.JZKn9ZT.B7f6/UcKHgJmXvZD6YpYzJfEd0JKW4T#pR@N2vZ3l+E8cL7Jm*d!1*K9gA#3uZc5W9!hM^%N7oLgVqF0JrTjWxQyUiYpA8hS4J6nOmAzKbX!Zr7"

// Reads the CSV file, in theory its agnostic to the origin of the file
func readExportFile(filename string) ([][]string, error) {
	fileData, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileData.Close()

	csvReader := csv.NewReader(fileData)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func goodReadsCSV(data [][]string) ([]extras.Book, error) {
	/*
		title - 1
		author - 2
		my rating - 7
		avg rating - 8
		page number - 11
		original publication - 13
		date read - 14
		date added - 15
		exclusive shelf - 18
		my review - 19
	*/
	books := make([]extras.Book, 0)
	//starts in 1 to avoid headers
	for i := 1; i < len(data); i++ {
		currentRow := data[i]
		books = append(books, extras.Book{
			Title:          currentRow[1],
			Author:         currentRow[2],
			MyRating:       extras.StringToFloatDefault(currentRow[7]),
			AVGRating:      extras.StringToFloatDefault(currentRow[8]),
			PageCount:      extras.StringToIntDefault(currentRow[11]),
			ReleaseYear:    extras.StringToIntDefault(currentRow[13]),
			FinishReading:  extras.StringToDateDefault(currentRow[14]),
			DateAdded:      extras.StringToDateDefault(currentRow[15]),
			Comment:        currentRow[19],
			TempCollection: currentRow[18],
		})
	}

	return books, nil
}

func getUserKey(token string) (string, error) {
	secretBytes := []byte(secret)
	decoded, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		return secretBytes, nil
	})
	if err != nil {
		return "", err
	}

	claims := decoded.Claims.(jwt.MapClaims)
	return claims["userKey"].(string), nil
}

func getUserAuth(email, password string) (*extras.AuthData, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	bodyBytes, err := makeRequest(false, "POST", "", "auth/login", "application/json", bytes.NewBuffer(body))
	token := string(bodyBytes)
	token = strings.ReplaceAll(token, "\"", "")
	token = strings.TrimSpace(token)

	userKey, err := getUserKey(token)
	if err != nil {
		return nil, err
	}

	return &extras.AuthData{
		Token:   token,
		Email:   email,
		UserKey: userKey,
	}, nil
}

func makeRequest(isAuth bool, method, token, url, content string, body io.Reader) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", baseurl, url), body)
	if err != nil {
		return nil, err
	}

	if isAuth {
		req.Header.Add("authorization", fmt.Sprintf("Bearer %s", token))
	}
	if body != nil {
		req.Header.Add("Content-Type", content)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Status != "200 OK" {
		fmt.Println(fmt.Sprint("Error en request ", resp.Status))
		return nil, errors.New("Bad request ")
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func getUserCollections(data *extras.AuthData, target *[]extras.Collection, wg *sync.WaitGroup, errchan chan (error)) {
	defer wg.Done()
	byteResponse, err := makeRequest(true, "GET", data.Token, fmt.Sprintf("collection/%s", data.UserKey), "", nil)
	if err != nil {
		errchan <- fmt.Errorf("Error inside collections routine %s", err.Error())
		return
	}

	err = json.Unmarshal(byteResponse, target)
	if err != nil {
		errchan <- fmt.Errorf("Error inside collections routine %s", err.Error())
		return
	}
}

func getAllLibrary(token string, target *[]extras.Book, wg *sync.WaitGroup, errchan chan (error)) {
	defer wg.Done()
	byteResponse, err := makeRequest(true, "GET", token, "admin/library", "", nil)
	if err != nil {
		errchan <- fmt.Errorf("Error inside library routine %s", err.Error())
		return
	}

	err = json.Unmarshal(byteResponse, target)
	if err != nil {
		errchan <- fmt.Errorf("Error inside library routine %s", err.Error())
		return
	}
}

// check todo list
func removeExistingBooks(library, importedBooks []extras.Book) []extras.Book {
	finalList := make([]extras.Book, 0)
	for j := 0; j < len(importedBooks); j++ {
		found := false
		for i := 0; i < len(library); i++ {
			if extras.MatchBooks(&importedBooks[j], &library[i]) {
				found = true
				break
			}
		}
		if !found {
			finalList = append(finalList, importedBooks[j])
		}
	}
	return finalList
}

func searchBook(importedBook *extras.Book, token string, wg *sync.WaitGroup, errChan chan (error), finalList *[]extras.Book, mu *sync.Mutex, failed *[]string) {
	defer wg.Done()
	body, _ := json.Marshal(map[string]string{
		"title": importedBook.Title,
	})
	bytes, err := makeRequest(true, "POST", token, "book/search", "application/json", bytes.NewBuffer(body))
	if err != nil {
		errChan <- fmt.Errorf("Error inside searching routine %s", err.Error())
		mu.Lock()
		*failed = append(*failed, importedBook.Title)
		mu.Unlock()
		return
	}
	posibleResults := make([]extras.Book, 0)
	err = json.Unmarshal(bytes, &posibleResults)
	if err != nil {
		errChan <- fmt.Errorf("Error inside searching routine %s", err.Error())
		mu.Lock()
		*failed = append(*failed, importedBook.Title)
		mu.Unlock()
		return
	}

	for i := 0; i < len(posibleResults); i++ {
		if extras.MatchBooks(importedBook, &posibleResults[i]) {
			posibleResults[i].TempCollection = importedBook.TempCollection
			mu.Lock()
			*finalList = append(*finalList, posibleResults[i])
			mu.Unlock()
			return
		}
	}

	mu.Lock()
	*failed = append(*failed, importedBook.Title)
	mu.Unlock()
	return
}

func writeToFile(filename string, data []byte) error {
	exported, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer exported.Close()
	_, err = exported.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	start := time.Now()

	data, err := getUserAuth("andresdglez@gmail.com", "An6248322")
	if err != nil {
		log.Fatal(err.Error())
	}

	records, err := readExportFile("export.csv")
	if err != nil {
		log.Fatal(err.Error())
	}
	importedBooks, err := goodReadsCSV(records)
	if err != nil {
		log.Fatal(err.Error())
	}

	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	var collections []extras.Collection
	var library []extras.Book

	wg.Add(2)
	go getAllLibrary(data.Token, &library, &wg, errChan)
	go getUserCollections(data, &collections, &wg, errChan)
	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	pendingBooks := removeExistingBooks(library, importedBooks)

	var booksToExport []extras.Book
	var failedBooks []string
	var wg2 sync.WaitGroup
	var mu sync.Mutex
	errChan2 := make(chan error, 1)

	for i := 0; i < len(pendingBooks); i++ {
		wg2.Add(1)
		go searchBook(&pendingBooks[i], data.Token, &wg2, errChan2, &booksToExport, &mu, &failedBooks)
	}
	wg2.Wait()
	close(errChan2)

	for err := range errChan2 {
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	toExport, _ := json.Marshal(booksToExport)
	failedFiles, _ := json.Marshal(failedBooks)
	writeToFile("export1", toExport)
	writeToFile("failed1", failedFiles)

	fmt.Println(time.Since(start).Seconds())
}
