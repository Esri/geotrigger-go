package geotrigger

import (
	"sync"
	"testing"
	"time"
)

func TestNewTokenManager(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)
	refute(t, tm, nil)
	expect(t, tm.getAccessToken(), "acc")
	expect(t, tm.getRefreshToken(), "rfr")

	tm.setAccessToken("merp")
	expect(t, tm.getAccessToken(), "merp")
}

func TestSimpleTokenRequest(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)

	// access token req
	tr := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}
	go tm.tokenRequest(tr)

	tokenResp := <-tr.tokenResponses
	refute(t, tokenResp, nil)
	expect(t, tokenResp.isAccessToken, true)
	expect(t, tokenResp.token, "acc")

	// refresh token req
	tr = &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}
	go tm.tokenRequest(tr)

	tokenResp = <-tr.tokenResponses
	refute(t, tokenResp, nil)
	expect(t, tokenResp.isAccessToken, false)
	expect(t, tokenResp.token, "rfr")
}

func TestUnknownPurposeInt(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)

	tr := &tokenRequest{
		purpose:        39846,
		tokenResponses: make(chan *tokenResponse),
	}
	go tm.tokenRequest(tr)

	tokenResp := <-tr.tokenResponses
	expect(t, tokenResp, nil)
}

func TestConcurrentTokenAccess(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)

	tr1 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr2 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr3 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr4 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr1)
	go tm.tokenRequest(tr2)
	go tm.tokenRequest(tr3)
	go tm.tokenRequest(tr4)

	var w sync.WaitGroup
	w.Add(4)
	go func() {
		tokenResp := <-tr1.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr2.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr3.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr4.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "acc")
		w.Done()
	}()
	w.Wait()
}

func TestTokenExpiry(t *testing.T) {
	// refresh succeeds first
	tm := newTokenManager("acc", "rfr", -100)

	tr1 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr2 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr3 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr4 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr1)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr1 arrives first
	go tm.tokenRequest(tr2)
	go tm.tokenRequest(tr3)
	go tm.tokenRequest(tr4)

	var w sync.WaitGroup
	w.Add(4)
	go func() {
		tokenResp := <-tr1.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	go func() {
		tokenResp := <-tr2.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr3.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr4.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	w.Wait()

	// refresh fails first
	tm.setExpiresAt(-100)
	tr5 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr6 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr7 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr8 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr5)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr5 arrives first
	go tm.tokenRequest(tr6)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr6 arrives second
	go tm.tokenRequest(tr7)
	go tm.tokenRequest(tr8)

	w.Add(4)
	go func() {
		tokenResp := <-tr5.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		rc := &tokenRequest{
			purpose:        refreshFailed,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	go func() {
		tokenResp := <-tr6.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}

		go tm.tokenRequest(rc)
		w.Done()
	}()
	go func() {
		tokenResp := <-tr7.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr8.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	w.Wait()
}

func TestMultipleRoutinesNeedRefreshToken(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)

	// refresh succeeds first
	tr1 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr2 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr3 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr4 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr1)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr1 arrives first
	go tm.tokenRequest(tr2)
	go tm.tokenRequest(tr3)
	go tm.tokenRequest(tr4)

	var w sync.WaitGroup
	w.Add(4)
	go func() {
		tokenResp := <-tr2.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr3.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr4.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr1.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	w.Wait()

	// refresh fails first
	tr5 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr6 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr7 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr8 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr5)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr5 arrives first
	go tm.tokenRequest(tr6)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr6 arrives second
	go tm.tokenRequest(tr7)
	go tm.tokenRequest(tr8)

	w.Add(4)
	go func() {
		tokenResp := <-tr7.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr8.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr6.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	go func() {
		tokenResp := <-tr5.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		rc := &tokenRequest{
			purpose:        refreshFailed,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	w.Wait()
}

func TestRefreshWithMultipleRoutinesNeedAccessToken(t *testing.T) {
	tm := newTokenManager("acc", "rfr", 1800)

	// refresh succeeds first
	tr1 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr2 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr3 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr4 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr1)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr1 arrives first
	go tm.tokenRequest(tr2)
	go tm.tokenRequest(tr3)
	go tm.tokenRequest(tr4)

	var w sync.WaitGroup
	w.Add(4)
	go func() {
		tokenResp := <-tr2.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr3.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr4.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr1.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	w.Wait()

	// refresh fails first
	tr5 := &tokenRequest{
		purpose:        refreshNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr6 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr7 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	tr8 := &tokenRequest{
		purpose:        accessNeeded,
		tokenResponses: make(chan *tokenResponse),
	}

	go tm.tokenRequest(tr5)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr5 arrives first
	go tm.tokenRequest(tr6)
	time.Sleep(20 * time.Millisecond) // pause to ensure tr6 arrives second
	go tm.tokenRequest(tr7)
	go tm.tokenRequest(tr8)

	w.Add(4)
	go func() {
		tokenResp := <-tr7.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr8.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, true)
		expect(t, tokenResp.token, "new acc")
		w.Done()
	}()
	go func() {
		tokenResp := <-tr6.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		tm.setAccessToken("new acc")
		rc := &tokenRequest{
			purpose:        refreshComplete,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	go func() {
		tokenResp := <-tr5.tokenResponses
		refute(t, tokenResp, nil)
		expect(t, tokenResp.isAccessToken, false)
		expect(t, tokenResp.token, "rfr")

		rc := &tokenRequest{
			purpose:        refreshFailed,
			tokenResponses: nil,
		}
		go tm.tokenRequest(rc)
		w.Done()
	}()
	w.Wait()
}
