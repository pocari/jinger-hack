package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sclevine/agouti"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/xerrors"
)

func withDriver(handler func(driver *agouti.WebDriver) error) error {
	options := []string{}
	if *headless {
		options = append(options, "--headless")
	}

	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", options),
	)
	if err := driver.Start(); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	defer driver.Stop()

	if err := handler(driver); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

func login(page *agouti.Page) error {
	if *email == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Email: ")
		readVal, err := reader.ReadString('\n')
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		*email = readVal
	}
	fmt.Print("Enter Password: ")
	passwordBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	fmt.Println()

	if err := page.Navigate("https://kintai.jinjer.biz/sign_in"); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if err := page.FindByName("company_code").Fill("2482"); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	if err := page.FindByName("email").Fill(strings.Trim(*email, " ")); err != nil {
		return xerrors.Errorf(": %w", err)
	}

	password := string(passwordBytes)
	if err := page.FindByName("password").Fill(strings.Trim(password, " ")); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	if err := page.FindByClass("login-button").Click(); err != nil {
		return xerrors.Errorf(": %w", err)
	}
	return nil
}
func genTimeCardUrl() (string, error) {
	if *month == "" {
		*month = time.Now().Format("2006-01")
	}

	return fmt.Sprintf("https://kintai.jinjer.biz/staffs/time_cards?month=%s", *month), nil
}

func hack() error {
	err := withDriver(func(d *agouti.WebDriver) error {
		page, err := d.NewPage(agouti.Browser("chrome"))
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		page.SetImplicitWait(10000)

		// ログイン
		// fmt.Println("login")
		if err := login(page); err != nil {
			return xerrors.Errorf(": %w", err)
		}

		if _, err := page.FindByLink("トップ").Visible(); err != nil {
			return xerrors.Errorf("ログインに失敗したかもしれません。: %w", err)
		}

		// fmt.Println("トップリンク待つ")
		page.FindByLink("トップ").Visible()
		// fmt.Println("トップリンクあった")
		// 一覧画面
		//fmt.Println("show time_cards")
		timeCardUrl, err := genTimeCardUrl()
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		fmt.Printf("%sの勤怠チェック\n", *month)
		if err := page.Navigate(timeCardUrl); err != nil {
			return xerrors.Errorf(": %w", err)
		}

		// fmt.Println("ブロックタイトル待つ")
		page.FindByClass("block_title").Visible()
		// fmt.Println("ブロックタイトルあった")

		// #main > div.jshopContainer > div.employee_table.scroll_margin.cf > div.table_wapper > table > tbody > tr:nth-child(11)
		dom, err := page.HTML()
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(dom))
		if err != nil {
			return xerrors.Errorf(": %w", err)
		}
		count := 0
		doc.Find("#main > div.jshopContainer > div.employee_table.scroll_margin.cf > div.table_wapper > table > tbody > tr").Each(func(_ int, s *goquery.Selection) {
			date := s.Find("td.date").Text()
			status := strings.Trim(s.Find("td.status > div.cf").Text(), " ")
			// holiday := strings.Trim(s.Find("td.holiday_td > div.cf").Text(), " ")
			if *all || status == "欠勤" {
				fmt.Printf("%s %s\n", date, status)
				count++
			}
		})

		if !*all && count == 0 {
			fmt.Println("欠勤無し!")
		}
		return nil
	})

	if err != nil {
		return xerrors.Errorf(": %w", err)
	}

	return nil
}

var (
	month    = flag.String("month", "", "指定された月を検索")
	all      = flag.Bool("all", false, "欠勤以外のステータスも")
	email    = flag.String("email", "", "ログイン用email")
	headless = flag.Bool("headless", true, "chromeを隠す場合true")
)

func main() {
	flag.Parse()

	err := hack()
	if err != nil {
		err := xerrors.Errorf(": %w", err)
		log.Printf("%+v\n", err)
	}
}
