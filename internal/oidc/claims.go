package oidc

import (
	"encoding/json"
	"fmt"
	"net/url"

	oauthelia2 "authelia.com/provider/oauth2"

	"github.com/authelia/authelia/v4/internal/model"
	"github.com/authelia/authelia/v4/internal/utils"
)

// NewClaimRequests parses the claims request parameter if set from a http.Request form.
func NewClaimRequests(form url.Values) (requests *ClaimsRequests, err error) {
	var raw string

	if raw = form.Get(FormParameterClaims); len(raw) == 0 {
		return nil, nil
	}

	requests = &ClaimsRequests{}

	if err = json.Unmarshal([]byte(raw), requests); err != nil {
		return nil, oauthelia2.ErrInvalidRequest.WithHint("The OAuth 2.0 client included a malformed 'claims' parameter in the authorization request.").WithWrap(err).WithDebugf("Error occurred attempting to parse the 'claims' parameter: %+v.", err)
	}

	return requests, nil
}

// ClaimsRequests is a request for a particular set of claims.
type ClaimsRequests struct {
	IDToken  map[string]*ClaimRequest `json:"id_token,omitempty"`
	UserInfo map[string]*ClaimRequest `json:"userinfo,omitempty"`
}

func (r *ClaimsRequests) GetIDTokenRequests() (requests map[string]*ClaimRequest) {
	if r == nil {
		return nil
	}

	return r.IDToken
}

func (r *ClaimsRequests) GetUserInfoRequests() (requests map[string]*ClaimRequest) {
	if r == nil {
		return nil
	}

	return r.UserInfo
}

func (r *ClaimsRequests) MatchesSubject(subject string) (requested string, ok bool) {
	if r == nil {
		return "", true
	}

	var request *ClaimRequest

	if r.UserInfo != nil {
		if request, ok = r.UserInfo[ClaimSubject]; ok {
			requested, ok = request.Value.(string)

			if request.Value != nil && request.Value != subject {
				return requested, false
			}
		}
	}

	if r.IDToken != nil {
		if request, ok = r.IDToken[ClaimSubject]; ok {
			requested, ok = request.Value.(string)

			if request.Value != nil && request.Value != subject {
				return requested, false
			}
		}
	}

	return requested, true
}

// ClaimRequest is a request for a particular claim.
type ClaimRequest struct {
	Essential bool  `json:"essential,omitempty"`
	Value     any   `json:"value,omitempty"`
	Values    []any `json:"values,omitempty"`
}

// Matches is a convenience function which tests if a particular value matches this claims request.
//
//nolint:gocyclo
func (r *ClaimRequest) Matches(value any) (match bool) {
	if r == nil {
		return false
	}

	switch t := value.(type) {
	case int:
		if r.Value != nil {
			if float64(t) != r.Value && t != r.Value {
				return false
			}
		}
	case int64:
		if r.Value != nil {
			if float64(t) != r.Value && t != r.Value {
				return false
			}
		}

		if r.Values != nil {
			found := false

			for _, v := range r.Values {
				if float64(t) == v || t == v {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}
	case float64:
		if r.Value != nil {
			if t != r.Value {
				return false
			}
		}

		if r.Values != nil {
			found := false

			for _, v := range r.Values {
				if t == v {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}
	case string:
		if r.Value != nil {
			if t != r.Value {
				return false
			}
		}

		if r.Values != nil {
			found := false

			for _, v := range r.Values {
				if t == v {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}
	case []string:
		if r.Value != nil {
			if !utils.IsStringInSlice(fmt.Sprintf("%s", value), t) {
				return false
			}
		}

		if r.Values != nil {
			found := false

		outer:
			for _, v := range r.Values {
				for _, w := range t {
					if v == w {
						found = true

						break outer
					}
				}
			}

			if !found {
				return false
			}
		}
	}

	return true
}

// GrantScopeAudienceConsent grants all scopes and audience values that have received consent.
func GrantScopeAudienceConsent(ar oauthelia2.AuthorizeRequester, consent *model.OAuth2ConsentSession) {
	if ar != nil {
		for _, scope := range consent.GrantedScopes {
			ar.GrantScope(scope)
		}

		for _, audience := range consent.GrantedAudience {
			ar.GrantAudience(audience)
		}
	}
}

// GrantClaims grants all claims the client is authorized to request.
func GrantClaims(strategy oauthelia2.ScopeStrategy, client Client, requests map[string]*ClaimRequest, detailer UserDetailer, extra map[string]any) {
	if requests == nil {
		return
	}

	scopes := client.GetScopes()

	for claim, request := range requests {
		switch claim {
		case ClaimGroups:
			grantScopeClaim(strategy, scopes, ScopeGroups, ClaimGroups, detailer.GetGroups(), request, extra)
		case ClaimPreferredUsername:
			grantScopeClaim(strategy, scopes, ScopeProfile, ClaimPreferredUsername, detailer.GetUsername(), request, extra)
		case ClaimFullName:
			grantScopeClaim(strategy, scopes, ScopeProfile, ClaimFullName, detailer.GetDisplayName(), request, extra)
		case ClaimPreferredEmail:
			emails := detailer.GetEmails()

			if len(emails) == 0 {
				continue
			}

			grantScopeClaim(strategy, scopes, ScopeEmail, ClaimPreferredEmail, emails[0], request, extra)
		case ClaimEmailAlts:
			emails := detailer.GetEmails()

			if len(emails) <= 1 {
				continue
			}

			grantScopeClaim(strategy, scopes, ScopeEmail, ClaimEmailAlts, emails[1:], request, extra)
		case ClaimEmailVerified:
			if !strategy(scopes, ScopeEmail) {
				continue
			}

			grantScopeClaim(strategy, scopes, ScopeEmail, ClaimEmailVerified, true, request, extra)
		}
	}
}

func grantScopeClaim(strategy oauthelia2.ScopeStrategy, scopes oauthelia2.Arguments, scope string, claim string, value any, request *ClaimRequest, extra map[string]any) {
	if !strategy(scopes, scope) {
		return
	}

	if request == nil || request.Value == nil || request.Values == nil {
		extra[claim] = value

		return
	}

	if request.Matches(value) {
		extra[claim] = value
	}
}