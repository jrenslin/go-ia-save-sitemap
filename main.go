
package main

import (
    "encoding/xml"
    "encoding/json"
    "errors"
    "io"
    "os"
    "fmt"
    "strconv"
    "log"
    "net/http"
    "time"
    "net/url"
)

type SitemapUrlset struct {
    Urls []SitemapUrl `xml:"url"`
}

type SitemapUrl struct {
    Loc string `xml:"loc"`
    Lastmod string `xml:"lastmod"`
}

type IaSnapshotClosest struct {
    Timestamp string `json:"timestamp"`
}

type IaArchivedSnapshots struct {
    Closest IaSnapshotClosest `json:"closest"`
}

type IaAvailabilityResponse struct {
    Url string `json:"url"`
    ArchivedSnapshots IaArchivedSnapshots `json:"archived_snapshots"`
}

const USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0"

func getRequestToByteArray(url string) ([]byte, error) {

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: 180 * time.Second,
    }

    // Create and modify HTTP request before sending
    request, err := http.NewRequest("GET", url, nil)
    if err != nil {
        log.Fatal(err)
    }
    request.Header.Set("User-Agent", USER_AGENT)

    // Make request
    response, err := client.Do(request)
    if err != nil {
        log.Fatal(err)
    }
    defer response.Body.Close()

    body, err := io.ReadAll(response.Body)
    if err != nil {
        log.Fatal(err)
    }

    return body, nil

}

func parseIaDate(strdate string) (time.Time, error) {

    if len(strdate) == 0 {
        return time.Now(), errors.New("Failed parsing time")
    }

    year, _ := strconv.Atoi(strdate[:4])
    month, _ := strconv.Atoi(strdate[4:6])
    day, _ := strconv.Atoi(strdate[6:8])
    hour, _ := strconv.Atoi(strdate[9:11])

    return time.Date(year, time.Month(month), day, hour, 1, 1, 1, time.UTC), nil
}

func iaSave(pageurl string) {

    fmt.Println("Will save url: " + pageurl)

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: 180 * time.Second,
    }

    // Create and modify HTTP request before sending
    request, err := http.NewRequest("GET", "https://web.archive.org/save/" + url.PathEscape(pageurl), nil)
    if err != nil {
        log.Fatal(err)
    }
    request.Header.Set("User-Agent", USER_AGENT)

    // Make request
    response, err := client.Do(request)
    if err != nil {
        log.Fatal(err)
    }
    defer response.Body.Close()

	fmt.Printf("client: got response!\n")
	fmt.Printf("client: status code: %d\n", response.StatusCode)

	resBody, parseerr := io.ReadAll(response.Body)
	if parseerr != nil {
		fmt.Printf("client: could not read response body: %s\n", parseerr)
		os.Exit(1)
	}
	fmt.Printf("Saved URL " + pageurl)
	fmt.Printf("client: response body: %s\n", resBody)

}

func iaSaveIfNecessary(sitemapUrl SitemapUrl) {

    body, err := getRequestToByteArray("https://archive.org/wayback/available?url=" + url.PathEscape(sitemapUrl.Loc))
    if err != nil {
        fmt.Println("Failed to get IA's availability info for:")
        fmt.Println(sitemapUrl)
        log.Fatal(err)
    }

    fmt.Println(sitemapUrl)
    var available IaAvailabilityResponse
    json.Unmarshal(body, &available)

    // If the snapshot is not available on IA, save the page
    fmt.Println(available.ArchivedSnapshots.Closest.Timestamp)
    lastAvailableTime, err := parseIaDate(available.ArchivedSnapshots.Closest.Timestamp)
    if err != nil {
        fmt.Println(sitemapUrl.Loc + " is not available on IA (failed to parse time).")
        iaSave(sitemapUrl.Loc)
        return
    }

    // If lastmod is not set, the saved page should not be older than half a year
    if sitemapUrl.Lastmod == "" {

        today := time.Now()
        lastYear := today.Add(-365 * 24 * time.Hour)


        if lastYear.After(lastAvailableTime) {
            fmt.Println(sitemapUrl.Loc + " is available on IA, but hasn't been saved for a year.")
            iaSave(sitemapUrl.Loc)
        }
        return
    }

    // If lastmod is given in the sitemap, and the page is
    // available on IA, check if the page is newer than the saved
    // snapshot
    lastModTime, err := time.Parse("2006-01-02", sitemapUrl.Lastmod)
	if err != nil {
		fmt.Println("Could not parse time of last modification:", err)
	}

    if lastModTime.After(lastAvailableTime) {
        fmt.Println(lastModTime)
        fmt.Println(lastAvailableTime)
        fmt.Println(sitemapUrl.Loc + " is available on IA, but was updated since. (" + lastModTime.String() + " > " +  lastAvailableTime.String() + ")")
        iaSave(sitemapUrl.Loc)
    } else {
        fmt.Println(sitemapUrl.Loc + " is available on IA and has not been updated since the last saving. (Last saved: " + lastAvailableTime.String() + ")")
    }

}

func getSitemapUrlFromCli() (string, error) {

    if (len(os.Args) < 2) {
        errMsg := "This tool loads a remote sitemap and archives all listed URLs in the Internet Archive' Wayback Machine (https://archive.org/).\n\nProvide a sitemap URL as the first command-line argument to get started."
        return "", errors.New(errMsg)
    }

    inputurl := os.Args[1]
    _, err := url.ParseRequestURI(inputurl)
    if (err != nil) {
        return "", err
    }

    return inputurl, nil

}

func main() {

    sitemapUrl, urlParseErr := getSitemapUrlFromCli()
    if (urlParseErr != nil) {
        fmt.Println(urlParseErr)
        os.Exit(0)
    }

    body, err := getRequestToByteArray(sitemapUrl)
    if err != nil {
        fmt.Println("Failed to get sitemap")
        log.Fatal(err)
    }

    var urlset SitemapUrlset
    xml.Unmarshal(body, &urlset)

    for _, url := range urlset.Urls {
        // Don't parallelize on purpose to save not overload IA
        iaSaveIfNecessary(url)
    }

}
