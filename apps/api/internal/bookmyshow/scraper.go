package bookmyshow

import (
	"context"
	"fmt"
	"time"

	"go-scraping/internal/movies"

	"github.com/chromedp/chromedp"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

type Scraper struct {
	timeout time.Duration
}

var _ movies.Scraper = (*Scraper)(nil)

func NewScraper(timeout time.Duration) *Scraper {
	return &Scraper{timeout: timeout}
}

func (s *Scraper) Scrape(ctx context.Context, city string) ([]movies.Movie, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(userAgent),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, s.timeout)
	defer cancel()

	url := fmt.Sprintf("https://in.bookmyshow.com/explore/movies-%s", city)
	selector := fmt.Sprintf("a[href*=\"/movies/%s/\"]", city)

	var links []map[string]string
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(fmt.Sprintf(`
			Array.from(document.querySelectorAll('%s')).map(link => {
				const h3Element = link.querySelector('h3');

				let title = '';
				if (h3Element) {
					title = h3Element.textContent.trim();
				} else {
					title = link.textContent.trim();
				}

				return {
					text: title,
					href: link.href
				};
			});
		`, selector), &links),
	)
	if err != nil {
		return nil, err
	}

	result := make([]movies.Movie, 0, len(links))
	for _, link := range links {
		href := link["href"]
		if href == "" {
			continue
		}

		result = append(result, movies.Movie{
			Title: movies.NormalizeQuery(link["text"]),
			Href:  href,
		})
	}

	return result, nil
}
