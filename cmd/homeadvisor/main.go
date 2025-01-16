package main

import (
	"errors"
	"fmt"
	"homeadvisorbot/core"
	"homeadvisorbot/entities"
	"homeadvisorbot/models"
	"homeadvisorbot/repositories"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/playwright-community/playwright-go"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v2"
)

const EXCEL_FILE_NAME = "Listings.xlsx"
const EXCEL_SHEET_NAME = "Sheet1"

type conf struct {
	SearchTerm       string `yaml:"search_term"`
	PageCountToCrawl int    `yaml:"page_count_to_crawl"`
}

func installBrowser() {
	err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"firefox"},
	})

	if err != nil {
		log.Fatal(err)
	}
}

func readFile(relativeFilePath string) string {
	workingDir, _ := os.Getwd()
	txt, err := os.ReadFile(workingDir + relativeFilePath)

	if err != nil {
		log.Println("error occurred while reading the file")
		panic(err)
	}

	return string(txt)
}

func addInitScript(browser *entities.BrowserType) {
	scriptTxt := readFile("\\resources\\scripts\\init.js")
	script := &playwright.Script{Content: &scriptTxt}

	err := (*browser.Page).AddInitScript(*script)

	if err != nil {
		log.Println("could not initscript to the page")
		panic(err)
	}
}

func makeDuckDuckGoSearch(browser *entities.BrowserType, searchTerm string) {
	browser.Navigate("https://lite.duckduckgo.com/lite/")
	browser.Fill("[class='query']", searchTerm)
	browser.Click("[value='Search']")
}

func getUrlsFromDuckDuckGoPage(browser *entities.BrowserType) []string {
	result, err := (*browser.Page).Locator("span.link-text").All()

	if err != nil {
		return []string{}
	}

	var urls []string

	for _, loc := range result {
		url, err := loc.InnerText()

		if err == nil && strings.HasPrefix(url, "www.") && strings.Contains(url, "/rated.") {
			urls = append(urls, "https://"+url)
		}
	}

	return urls
}

func extractResultCount(resultString string) (int, error) {
	re := regexp.MustCompile(`Showing 1-\d+ of (\d+) results`)
	match := re.FindStringSubmatch(resultString)

	if len(match) == 0 {
		return 0, errors.New("could not extract result count")
	}

	resultCountStr := match[1]

	resultCount, err := strconv.Atoi(resultCountStr)
	if err != nil {
		return 0, errors.New("invalid result count")
	}

	return resultCount, nil
}

func scrapeHomeadvisorUrl(homeadvisorRepository *repositories.HomeadvisorRepository, browser *entities.BrowserType, url string) *models.HomeadvisorListingModel {
	var homeadvisorListing models.HomeadvisorListingModel

	browser.Navigate(url)

	err := browser.Click(".company-details-card > div:nth-child(2) > div:nth-child(2)")

	if err != nil {
		log.Fatal("could not open company details card")
	}

	var phoneNumber string

	phoneNumberLocator := (*browser.Page).Locator(".contact-container > div:nth-child(2) > a:nth-child(1)")
	phoneNumberElIsVisible, err := phoneNumberLocator.IsVisible()

	if err != nil || !phoneNumberElIsVisible {
		log.Println("could not get company phone number, might be missing from the list")
	} else {
		browser.Click(".contact-container > div:nth-child(2) > a:nth-child(1)")
		phoneNumber, _ = (*browser.Page).Locator(".contact-container > div:nth-child(2) > a:nth-child(2)").First().InnerText()
	}

	pageUrl := (*browser.Page).URL()
	companyUrlSelector := ".contact-container > a:nth-child(4)"

	if len(phoneNumber) == 0 {
		companyUrlSelector = ".contact-container > a"
	}

	companyUrl, err := (*browser.Page).Locator(companyUrlSelector).First().InnerText()

	if err != nil {
		log.Println("could not get the company url, might be missing from the listing")
	}

	companyName, err := (*browser.Page).Locator("[class='@w-full @text-3xl']").First().InnerText()

	if err != nil {
		log.Println("could not get the company name")
	}

	state, err := (*browser.Page).Locator("#breadcrumbs > div:nth-child(1) > span:nth-child(2) > a:nth-child(2)").First().InnerText()

	if err != nil {
		log.Println("ould not get the company state")
	}

	city, err := (*browser.Page).Locator("#breadcrumbs > div:nth-child(1) > span:nth-child(3) > a:nth-child(2)").First().InnerText()

	if err != nil {
		log.Println("could not get the company city")
	}

	var reviewCount int
	reviewCountWrappedTxt, err := (*browser.Page).Locator("[class=\"@pr-4 @order-2 md:@order-1\"] > span").First().InnerText()

	if err != nil {
		log.Println("could not find review count within the website")
	} else {
		reviewCount, err = extractResultCount(reviewCountWrappedTxt)

		if err != nil {
			log.Println("could not extract the review count, entering 0")
		}
	}

	var lastReviewDate time.Time

	if reviewCount > 0 {
		lastReviewDateStr, err := (*browser.Page).Locator("[class='@flex-initial @text-gray @font-semibold md:@self-end']").First().InnerText()

		if err != nil {
			log.Println("could not get a last review date")
		} else {
			dateLayout := "1/2/2006"
			lastReviewDate, err = time.Parse(dateLayout, lastReviewDateStr)

			if err != nil {
				log.Println("error parsing date ", err)
			}
		}
	}

	homeadvisorListing = models.HomeadvisorListingModel{
		Url:            pageUrl,
		CompanyName:    &companyName,
		PhoneNumber:    &phoneNumber,
		CompanyUrl:     &companyUrl,
		State:          &state,
		City:           &city,
		LastReviewDate: &lastReviewDate,
		ReviewCount:    uint(reviewCount),
	}

	homeadvisorRepository.Create(&homeadvisorListing)
	log.Println("saved a new listing to the db")

	return &homeadvisorListing
}

func getConf() *conf {
	c := &conf{}

	workingDir, _ := os.Getwd()
	yamlFile, err := os.ReadFile(workingDir + "\\config.yaml")

	if err != nil {
		log.Println("couldnt open conf file")
	}

	err = yaml.Unmarshal(yamlFile, c)

	if err != nil {
		log.Println("couldnt read conf file")
		panic(err)
	}

	return c
}

func main() {
	c := getConf()

	db := core.NewDatabase()
	homeadvisorRepository := repositories.NewHomeadvisorRepository(db)

	installBrowser()
	pw, err := playwright.Run()

	if err != nil {
		panic(err)
	}

	headless := true

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: &headless,
	}

	browser, err := pw.Firefox.Launch(launchOptions)

	if err != nil {
		log.Fatal(err)
	}

	page, err := browser.NewPage()

	if err != nil {
		log.Fatal(err)
	}

	fBrowser := entities.NewBrowser(&browser, &page)
	addInitScript(fBrowser)
	currentDate := time.Now()

	dateString := currentDate.Format("2/1/2006")

	makeDuckDuckGoSearch(fBrowser, dateString+" "+c.SearchTerm)

	if err != nil {
		log.Fatal(err)
	}

	var urls []string

	urlsInPage := getUrlsFromDuckDuckGoPage(fBrowser)
	urls = append(urls, urlsInPage...)

	for currentPage := 1; currentPage <= c.PageCountToCrawl; currentPage++ {
		fBrowser.Click("[type='submit'][value='Next Page >']")
		(*fBrowser.Page).WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle})

		urlsInPage := getUrlsFromDuckDuckGoPage(fBrowser)
		urls = append(urls, urlsInPage...)
	}

	for _, url := range urls {
		log.Println(url)
		err, exists := homeadvisorRepository.DoesItExist(url)

		if err != nil {
			log.Println(err)
			continue
		}

		if !exists {
			scrapeHomeadvisorUrl(homeadvisorRepository, fBrowser, url)
		}
	}

	(*fBrowser).Dispose()

	saveListingsToExcelFile(homeadvisorRepository)

	log.Println("task finished")
}

func createExcelFile() {
	f := excelize.NewFile()

	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()

	headers := []string{"URL", "Company URL", "Company Name", "Phone Number", "State", "City", "Review Count", "Last Review Date"}
	for i, header := range headers {
		f.SetCellValue(EXCEL_SHEET_NAME, fmt.Sprintf("%s%d", string(rune(65+i)), 1), header)
	}

	if err := f.SaveAs(EXCEL_FILE_NAME); err != nil {
		log.Println(err)
	}
}

func openExcelFile() *excelize.File {
	f, err := excelize.OpenFile(EXCEL_FILE_NAME)

	if err != nil {
		log.Println(err)
		return nil
	}

	return f
}

func safeDereference(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func saveListingsToExcelFile(homeadvisorRepository *repositories.HomeadvisorRepository) {
	log.Println("saving listings to excel file")
	listings := homeadvisorRepository.All()

	createExcelFile()
	f := openExcelFile()

	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()

	var data [][]interface{}

	for _, listing := range listings {
		lInterface := []interface{}{
			listing.Url,
			safeDereference(listing.CompanyUrl),
			safeDereference(listing.CompanyName),
			safeDereference(listing.PhoneNumber),
			safeDereference(listing.State),
			safeDereference(listing.City),
			listing.ReviewCount,
			listing.LastReviewDate,
		}

		data = append(data, lInterface)
	}

	for idx, row := range data {
		cell, err := excelize.CoordinatesToCellName(1, idx+2)

		if err != nil {
			log.Println(err)
			return
		}

		f.SetSheetRow(EXCEL_SHEET_NAME, cell, &row)
	}

	cols, _ := f.GetCols(EXCEL_SHEET_NAME)

	for idx, col := range cols {
		largestWidth := 0
		for _, rowCell := range col {
			cellWidth := utf8.RuneCountInString(rowCell) + 2
			if cellWidth > largestWidth {
				largestWidth = cellWidth
			}
		}
		name, _ := excelize.ColumnNumberToName(idx + 1)
		f.SetColWidth(EXCEL_SHEET_NAME, name, name, float64(largestWidth))
	}

	if err := f.SaveAs(EXCEL_FILE_NAME); err != nil {
		log.Println(err)
	} else {
		log.Println("saved listings to excel file")
	}
}
