package main

import (
    "bufio"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"
)

// ANSI color codes for formatting
const (
    RedColor   = "\033[91m"
    ResetColor = "\033[0m"
)

var (
    totalTargets  int
    loadedTargets int
    counterMutex  sync.Mutex
    searchWord    string
    baseURL       string
    domainList    string
)

func main() {
    flag.StringVar(&baseURL, "u", "", "Base URL with FUZZ placeholder, e.g., https://FUZZ.com/admin.php")
    flag.StringVar(&searchWord, "w", "FUZZ", "Specify the word to search for in the response body.")
    flag.StringVar(&domainList, "l", "", "Path to the file containing the list of URLs.")
    flag.Parse()

    if domainList == "" {
        fmt.Println("Usage: ./main -l <file_path> -w <word>")
        os.Exit(1)
    }

    urls, err := readURLsFromFile(domainList)
    if err != nil {
        fmt.Println("Error reading URLs from the file:", err)
        os.Exit(1)
    }

    totalTargets = len(urls)
    client := &http.Client{Timeout: 5 * time.Second}

    // Create channels for communication between main and worker goroutines
    urlsChannel := make(chan string, totalTargets)
    resultsChannel := make(chan string, totalTargets)
    done := make(chan bool)

    // Start the worker pool with 40 goroutines
    go startWorkerPool(urlsChannel, resultsChannel, 100, client)

    // Start the results processor
    go processResults(resultsChannel, done)

    // Feed URLs to the worker pool
    for _, url := range urls {
        incrementCounter()
        urlsChannel <- url
    }
    close(urlsChannel)

    // Wait for all workers to finish
    <-done
}

func startWorkerPool(urls <-chan string, results chan<- string, numWorkers int, client *http.Client) {
    var wg sync.WaitGroup

    // Create worker goroutines
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            processURLs(urls, results, client)
        }()
    }

    // Close the results channel when all workers are done
    go func() {
        wg.Wait()
        close(results)
    }()
}

func processURLs(urls <-chan string, results chan<- string, client *http.Client) {
    for url := range urls {
        resp, err := client.Get(url)
        if err != nil {
            // Handle errors
            continue
        }
        defer resp.Body.Close()

        // Read the response body with a timeout
        body, readErr := readResponseBodyWithTimeout(resp.Body, 2*time.Second)
        if readErr != nil {
            // Handle read errors
            continue
        }

        contentType := resp.Header.Get("Content-Type")
        if strings.Contains(string(body), searchWord) {
            results <- fmt.Sprintf("[*][%s]%s[ POSSIBLE XSS ! ]%s [ %s ]\n", url, RedColor, ResetColor, contentType)
        }
    }
}

func processResults(results <-chan string, done chan<- bool) {
    for result := range results {
        fmt.Print(result)
    }
    done <- true
}

func readURLsFromFile(filePath string) ([]string, error) {
    var urls []string

    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        urlsInLine := strings.FieldsFunc(line, func(r rune) bool {
            return r == '\r' || r == '\n' || r == '\t'
        })
        urls = append(urls, urlsInLine...)
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return urls, nil
}

func incrementCounter() {
    counterMutex.Lock()
    loadedTargets++
    counterMutex.Unlock()
}

func readResponseBodyWithTimeout(body io.Reader, timeout time.Duration) ([]byte, error) {
    done := make(chan struct{})
    var result []byte
    var err error

    go func() {
        defer close(done)
        result, err = io.ReadAll(body)
    }()

    select {
    case <-done:
        return result, err
    case <-time.After(timeout):
        return nil, fmt.Errorf("context deadline exceeded (Client.Timeout or context cancellation while reading body)")
    }
}
