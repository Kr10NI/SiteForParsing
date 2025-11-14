package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type ParseRequest struct {
	URL       string   `json:"url"`
	Selectors []string `json:"selectors"`
}

type ParseResult struct {
	Selector string   `json:"selector"`
	Results  []string `json:"results"`
}

type ParseResponse struct {
	Results []ParseResult `json:"results"`
}

var dynamicSites = []string{
	"sportmaster.ru",
	"wildberries.ru",
	"ozon.ru",
	"aliexpress.com",
	"lamoda.ru",
}

func isDynamicSite(url string) bool {
	for _, domain := range dynamicSites {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

func parseHandler(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Запрос: url=%s, selectors=%v", req.URL, req.Selectors)

	useChromedp := isDynamicSite(req.URL)

	var html string
	if useChromedp {
		if strings.Contains(req.URL, "sportmaster.ru") {
			apiUrl := "https://www.sportmaster.ru/api/catalog/product/?slug=futbolki&categoryId=1000000000001973639"
			log.Printf("XHR: GET %s", apiUrl)
			client := &http.Client{}
			reqApi, err := http.NewRequest("GET", apiUrl, nil)
			if err != nil {
				http.Error(w, "Failed to create XHR API request", http.StatusInternalServerError)
				log.Printf("XHR request error: %v", err)
				return
			}
			reqApi.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			reqApi.Header.Set("Referer", "https://www.sportmaster.ru/")
			reqApi.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
			reqApi.Header.Set("X-Requested-With", "XMLHttpRequest")
			// reqApi.Header.Set("Cookie", "smid=...; smcid=...; _ym_uid=...; _ym_d=...; _gid=...; _ga=...; device_id=...; sessionid=...")
			resp, err := client.Do(reqApi)
			if err != nil {
				http.Error(w, "Failed to fetch XHR API", http.StatusInternalServerError)
				log.Printf("XHR error: %v", err)
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(w, "Failed to read XHR API response", http.StatusInternalServerError)
				log.Printf("XHR read error: %v", err)
				return
			}
			log.Printf("XHR: получено %d байт", len(body))
			log.Printf("XHR JSON body: %s", string(body))

			results := []ParseResult{{Selector: "sportmaster_raw_json", Results: []string{string(body)}}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ParseResponse{Results: results})
			return
		}

		waitSelector := ".sm-product-card__info"
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		log.Printf("chromedp: стартуем парсинг %s, ждем %s", req.URL, waitSelector)
		err := chromedp.Run(ctx,
			chromedp.Navigate(req.URL),
			chromedp.WaitVisible(waitSelector, chromedp.ByQuery),
			chromedp.OuterHTML("html", &html),
		)
		log.Printf("chromedp: завершено, длина html: %d, ошибка: %v", len(html), err)
		if err != nil {
			http.Error(w, "chromedp error", http.StatusInternalServerError)
			log.Printf("chromedp error: %v", err)
			return
		}
	} else {
		client := &http.Client{}
		reqHttp, err := http.NewRequest("GET", req.URL, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
		reqHttp.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		resp, err := client.Do(reqHttp)
		if err != nil {
			http.Error(w, "Failed to fetch URL", http.StatusInternalServerError)
			log.Printf("Ошибка HTTP-запроса: %v", err)
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			log.Printf("Ошибка чтения тела ответа: %v", err)
			return
		}
		log.Println(string(body))
		html = string(body)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		http.Error(w, "Failed to parse HTML", http.StatusInternalServerError)
		log.Printf("Ошибка парсинга HTML: %v", err)
		return
	}
	var results []ParseResult
	for _, sel := range req.Selectors {
		var selResults []string
		doc.Find(sel).Each(func(i int, s *goquery.Selection) {
			fullText := s.Text()
			if fullText != "" {
				selResults = append(selResults, fullText)
			}
		})
		results = append(results, ParseResult{Selector: sel, Results: selResults})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ParseResponse{Results: results})
}

func main() {
	http.HandleFunc("/api/parse", parseHandler)
	log.Println("Go backend running on :9095")
	log.Fatal(http.ListenAndServe(":9095", nil))
}
