package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type VideoLink struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Topic     string    `json:"topic"`
	Sign      int       `json:"sign"`
	SignBasis string    `json:"sign_basis"`
	RuleIDs   []string  `json:"rule_ids"`
	Enabled   bool      `json:"enabled"`
	Version   int       `json:"version"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

var videoLinkSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var errVideoLinkConflict = errors.New("Video link changed; reload before saving. Use a different slug for a new link.")
var errVideoLinkRules = errors.New("A selected reading is unavailable or retired. Choose current approved readings before publishing.")
var errVideoLinkLimit = errors.New("The 500 video link limit has been reached.")

// validVideoSlug restricts stable public link IDs to short path-safe lowercase slugs.
func validVideoSlug(slug string) bool {
	return len(slug) >= 3 && len(slug) <= 64 && videoLinkSlug.MatchString(slug)
}

// validateVideoLink bounds the single public title and accepts only declared topics, sign references and unique reading IDs.
func validateVideoLink(v VideoLink) error {
	if !validVideoSlug(v.Slug) || utf8.RuneCountInString(strings.TrimSpace(v.Title)) < 3 || utf8.RuneCountInString(v.Title) > 180 || len(v.RuleIDs) > 40 || v.Version < 0 {
		return fmt.Errorf("Enter a valid slug and title (3–180 characters). Select at most 40 readings.")
	}
	switch v.Topic {
	case "debt", "promotion", "zodiac", "general", "personality", "education", "career", "relationships", "family", "wellbeing", "dasha":
	default:
		return fmt.Errorf("Choose a supported video topic.")
	}
	if v.SignBasis != "moon" && v.SignBasis != "lagna" && v.SignBasis != "sun" {
		return fmt.Errorf("Choose Moon sign, ascendant or Sun sign.")
	}
	if v.Sign < 0 || v.Sign > 12 || (v.Topic == "zodiac" && v.Sign == 0) || (v.Topic != "zodiac" && v.Sign != 0) {
		return fmt.Errorf("Choose a zodiac sign for zodiac videos only.")
	}
	seen := map[string]bool{}
	for _, id := range v.RuleIDs {
		if id == "" || len(id) > 80 || seen[id] {
			return fmt.Errorf("Choose unique interpretation IDs.")
		}
		seen[id] = true
	}
	return nil
}

// latestVideoRules offers only the latest approved, non-demo interpretation of each ID, honoring later retirement.
func latestVideoRules(rules map[string]Rule) []Rule {
	latest := map[string]Rule{}
	for _, rule := range rules {
		if rule.Demo || (rule.Status != "approved" && rule.Status != "retired") {
			continue
		}
		if old, ok := latest[rule.ID]; !ok || rule.Version > old.Version {
			latest[rule.ID] = rule
		}
	}
	out := []Rule{}
	for _, rule := range latest {
		if rule.Status == "approved" && rule.Approver != "" && rule.ApprovedAt != nil {
			out = append(out, rule)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// publicVideoLink exposes one active campaign without returning draft links, customer records or unapproved interpretations.
func (a *App) publicVideoLink(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !validVideoSlug(slug) {
		fail(w, 404, fmt.Errorf("This video link is unavailable."))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var link VideoLink
	found := false
	if err := a.Store.ViewKinds(ctx, []string{"video_links"}, func(s *State) error {
		link, found = s.VideoLinks[slug]
		return nil
	}); err != nil {
		fail(w, 503, fmt.Errorf("Video link unavailable. Please try again."))
		return
	}
	if !found || !link.Enabled {
		fail(w, 404, fmt.Errorf("This video link is unavailable."))
		return
	}
	jsonOut(w, link)
}

// adminVideoLinks lists link drafts and eligible interpretation choices behind the separate administrator session.
func (a *App) adminVideoLinks(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	links := []VideoLink{}
	rules := []Rule{}
	if err := a.Store.ViewKinds(ctx, []string{"video_links", "rules"}, func(s *State) error {
		for _, link := range s.VideoLinks {
			links = append(links, link)
		}
		rules = latestVideoRules(s.Rules)
		return nil
	}); err != nil {
		fail(w, 503, fmt.Errorf("Video links unavailable. Please try again."))
		return
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].Updated.Equal(links[j].Updated) {
			return links[i].Slug < links[j].Slug
		}
		return links[i].Updated.After(links[j].Updated)
	})
	jsonOut(w, map[string]any{"links": links, "rules": rules})
}

// saveVideoLink atomically saves a validated link with optimistic version checks, approved rule references and an audit entry.
func (a *App) saveVideoLink(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in VideoLink
	if !decode(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Slug != r.PathValue("slug") {
		fail(w, 422, fmt.Errorf("The link slug must match the URL."))
		return
	}
	if err := validateVideoLink(in); err != nil {
		fail(w, 422, err)
		return
	}
	expectedVersion := in.Version
	err := a.Store.Update(r.Context(), func(s *State) error {
		old, exists := s.VideoLinks[in.Slug]
		if (!exists && expectedVersion != 0) || (exists && expectedVersion != old.Version) {
			return errVideoLinkConflict
		}
		if !exists && len(s.VideoLinks) >= 500 {
			return errVideoLinkLimit
		}
		approved := map[string]bool{}
		for _, rule := range latestVideoRules(s.Rules) {
			approved[rule.ID] = true
		}
		if in.Enabled {
			for _, id := range in.RuleIDs {
				if !approved[id] {
					return errVideoLinkRules
				}
			}
		}
		if exists {
			in.Created = old.Created
		} else {
			in.Created = a.Now()
		}
		in.Version = old.Version + 1
		in.Updated = a.Now()
		if in.RuleIDs == nil {
			in.RuleIDs = []string{}
		}
		if s.VideoLinks == nil {
			s.VideoLinks = map[string]VideoLink{}
		}
		s.VideoLinks[in.Slug] = in
		audit(s, sess.UserID, "video-link.saved", in.Slug)
		return nil
	})
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, errVideoLinkConflict) {
			status = http.StatusConflict
		}
		// Storage errors remain retryable without exposing provider details.
		if !errors.Is(err, errVideoLinkConflict) && !errors.Is(err, errVideoLinkRules) && !errors.Is(err, errVideoLinkLimit) {
			fail(w, 503, fmt.Errorf("Video link could not be saved. Please try again."))
			return
		}
		fail(w, status, err)
		return
	}
	jsonOut(w, in)
}
