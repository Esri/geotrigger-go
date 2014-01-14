package geotrigger_golang

import "fmt"

type TokenManager interface {
	// the manageTokens() func should loop in a routine and manage
	// access to the tokens for other routines
	manageTokens()
	// access tokens in a threadsafe way, but with possible wait time
	tokenRequest(*tokenRequest)
	// normal getters for immediate *unsafe* access
	getAccessToken() string
	getRefreshToken() string
	// used safely when refreshing the access token
	setAccessToken(string)
}

type tokenManager struct {
	tokenRequests chan *tokenRequest
	accessToken   string
	refreshToken  string
}

/* consts and structs for channel coordination */
const (
	accessNeeded = iota
	refreshNeeded
	refreshComplete
	refreshFailed
)

type tokenRequest struct {
	purpose        int
	tokenResponses chan *tokenResponse
}

type tokenResponse struct {
	token         string
	isAccessToken bool
}

func newTokenManager(accessToken string, refreshToken string) TokenManager {
	return &tokenManager{
		tokenRequests: make(chan *tokenRequest),
		accessToken:   accessToken,
		refreshToken:  refreshToken,
	}
}

func newTokenRequest(purpose int, makeChan bool) *tokenRequest {
	var responses chan *tokenResponse
	if makeChan {
		responses = make(chan *tokenResponse)
	}

	return &tokenRequest{
		purpose:        purpose,
		tokenResponses: responses,
	}
}

func (tm *tokenManager) tokenRequest(tr *tokenRequest) {
	tm.tokenRequests <- tr
}

func (tm *tokenManager) getAccessToken() string {
	return tm.accessToken
}

func (tm *tokenManager) getRefreshToken() string {
	return tm.refreshToken
}

func (tm *tokenManager) setAccessToken(token string) {
	tm.accessToken = token
}

func (tm *tokenManager) manageTokens() {
	var waitingRequests []*tokenRequest
	refreshInProgress := false
	for {
		tr := <-tm.tokenRequests

		switch {
		case tr.purpose == refreshFailed:
			nextRequest := waitingRequests[0]
			waitingRequests = waitingRequests[1:]

			if nextRequest.purpose == refreshNeeded {
				refreshInProgress = true
				go tokenApproved(nextRequest, tm.refreshToken, false)
			} else if nextRequest.purpose == accessNeeded {
				refreshInProgress = false
				go tokenApproved(nextRequest, tm.accessToken, true)
			}
		case tr.purpose == refreshComplete:
			if !refreshInProgress {
				fmt.Println("Warning: refresh completed when we assumed none were occurring.")
			}
			refreshInProgress = false

			// copy waiting token requests to new slice for iterating
			currentWaitingReqs := make([]*tokenRequest, len(waitingRequests))
			copy(currentWaitingReqs, waitingRequests)

			// clear main status checks slice (as we might get more added shortly)
			waitingRequests = waitingRequests[:0]

			for _, waitingReq := range currentWaitingReqs {
				go tokenApproved(waitingReq, tm.accessToken, true)
			}
		case refreshInProgress:
			waitingRequests = append(waitingRequests, tr)
		case tr.purpose == refreshNeeded:
			refreshInProgress = true
			go tokenApproved(tr, tm.refreshToken, false)
		case tr.purpose == accessNeeded:
			go tokenApproved(tr, tm.accessToken, true)
		}
	}
}

func tokenApproved(tr *tokenRequest, token string, isAccessToken bool) {
	tr.tokenResponses <- &tokenResponse{
		token:         token,
		isAccessToken: isAccessToken,
	}
}
