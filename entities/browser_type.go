package entities

import (
	"errors"
	"log"

	"github.com/playwright-community/playwright-go"
)

type BrowserType struct {
	Browser *playwright.Browser
	Page    *playwright.Page
}

func (b *BrowserType) Navigate(url string) {
	_, err := (*b.Page).Goto(url)
	(*b.Page).WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateNetworkidle})

	if err != nil {
		log.Println("err while navigating to url " + url)
		panic(err)
	}
}

func (b *BrowserType) Fill(selector string, text string) {
	err := (*b.Page).Locator(selector).Fill(text)

	if err != nil {
		log.Println("err occurred while filling selector " + selector)
		panic(err)
	}
}

func (b *BrowserType) Click(selector string) error {
	maxAttempts := 5
	timeout := 15000.0

	for i := range maxAttempts + 1 {
		err := (*b.Page).Locator(selector).First().Click(playwright.LocatorClickOptions{
			Timeout: &timeout,
		})

		if err != nil {
			log.Println("error occurred while clicking to selector " + selector)
			log.Println(err)

			if i < maxAttempts {
				(*b.Page).Reload()
				log.Println("retrying...")
				return b.Click(selector)
			} else {
				return errors.New("failed clicking to an element")
			}
		} else {
			return nil
		}
	}

	return nil
}

func (b *BrowserType) Dispose() {
	log.Println("disposing the browser")

	(*b.Page).Close()
	(*b.Browser).Close()
}

func NewBrowser(browser *playwright.Browser, page *playwright.Page) *BrowserType {
	return &BrowserType{
		Browser: browser,
		Page:    page,
	}
}
