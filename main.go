package chatgpt

import (
	"context"
	"fmt"
	"net/http"
	"time"

	cu "github.com/Davincible/chromedp-undetected"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type TokenGen struct {
	cancel context.CancelFunc
	ctx    context.Context
}

func NewTokenGen() *TokenGen {
	return &TokenGen{}
}

func (t *TokenGen) StartChrome() (context.Context, error) {
	ctx, tcancel, err := cu.New(cu.NewConfig(
		// cu.WithHeadless(),
		cu.WithUserDataDir("./dp"),
		cu.WithTimeout(10*time.Second),
	))
	if err != nil {
		return nil, err
	}
	t.cancel = tcancel
	t.ctx = ctx
	return ctx, nil
}

func (t *TokenGen) CloseChrome() {
	t.cancel()
}

func (t *TokenGen) GetToken(email, password string) (string, error) {
	var cookiesX []*http.Cookie
	if err := chromedp.Run(t.ctx,
		chromedp.Navigate("https://chat.openai.com/auth/login"),
		chromedp.Click(`//*[@id="__next"]/div[1]/div[1]/div[4]/button[1]`),
		chromedp.WaitVisible(`//*[@id="username"]`),
		chromedp.SendKeys(`//*[@id="username"]`, email),
		chromedp.Click(`/html/body/div/main/section/div/div/div/div[1]/div/form/div[2]/button`),
		chromedp.WaitVisible(`//*[@id="password"]`),
		chromedp.SendKeys(`//*[@id="password"]`, password),
		chromedp.Click(`/html/body/div/main/section/div/div/div/form/div[3]/button`),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)

			if err != nil {
				return err
			}

			for _, cookie := range cookies {
				cookiesX = append(cookiesX, &http.Cookie{
					Name:  cookie.Name,
					Value: cookie.Value,
				})
			}
			return nil
		}),

		// next page is json, display it rather than autodownload : net err

	); err != nil {
		return "", err
	}

	// balace TODO: check if token is valid

	for _, cookie := range cookiesX {
		if cookie.Name == "__Secure-next-auth.session-token" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("token not found")
}

func main() {
	t := NewTokenGen()
	_, err := t.StartChrome()
	if err != nil {
		panic(err)
	}

	tok, err := t.GetToken("", "")
	if err != nil {
		panic(err)
	}

	fmt.Println(tok)

}
