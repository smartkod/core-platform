package service

// ads.go implements econ.Ads: rewarded-ad serving, server-stamped watch
// sessions, clicks and the admin surface.
//
// Correctness rules (L2.md rev 1.1 §3.11):
//   - required_seconds and the session stamps are server facts; the client
//     cannot influence when a claim becomes payable (REWARD_TOO_EARLY).
//   - one earning session per (user, ad, day), enforced by a unique index, so
//     farming a single ad is structurally impossible.
//   - watch and click rewards draw from one pool: the daily/monthly caps are
//     checked on the ledger sum of ad_watch_reward + ad_click_reward entries
//     (decision E13), always over UTC day/month boundaries.
//   - every payout is one ledger movement with an idempotency key derived from
//     the session or click identity, so a retry never pays twice.

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/model"
	"core-platform/mod/economy/repository"
)

// Ads is the rewarded-ads service facade implementing econ.Ads.
type Ads struct {
	wallet *Wallet
	store  *repository.Store
	now    func() time.Time
}

// NewAds builds the ads service over the wallet (its payout funnel) and the
// shared repository store.
func NewAds(wallet *Wallet, store *repository.Store, now func() time.Time) *Ads {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Ads{wallet: wallet, store: store, now: now}
}

// Watch-session expiry: the window closes at min(2x watch_seconds, 5m) but
// never before the required duration itself (a short ad still gets its full
// time to finish; a long one cannot keep a session open indefinitely).
func sessionExpiry(start time.Time, requiredSeconds int) time.Time {
	d := time.Duration(2*requiredSeconds) * time.Second
	if d > 5*time.Minute {
		d = 5 * time.Minute
	}
	if d < time.Duration(requiredSeconds)*time.Second {
		d = time.Duration(requiredSeconds) * time.Second
	}
	return start.Add(d)
}

// utcDay and utcMonthStart are the cap windows (UTC, L2.md open question A6).
// They mirror repository.today/firstOfMonth; keep both in sync.
func utcDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func utcMonthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func nextDayStart(now time.Time) time.Time {
	day := utcDay(now)
	return day.AddDate(0, 0, 1)
}

func nextMonthStart(now time.Time) time.Time {
	m := utcMonthStart(now)
	return m.AddDate(0, 1, 0)
}

// earned loads both cap totals for the user in one pass.
func (a *Ads) earnedInTx(tx *gorm.DB, userID, currency string, now time.Time) (daily, monthly int64, err error) {
	reasons := econ.UserEarnRelatedReasons
	daily, err = a.store.EarnedSince(tx, userID, currency, reasons, utcDay(now))
	if err != nil {
		return 0, 0, err
	}
	monthly, err = a.store.EarnedSince(tx, userID, currency, reasons, utcMonthStart(now))
	return daily, monthly, err
}

// budgetOf computes the dashboard budget from earned totals and the dials.
func budgetOf(settings econ.RewardSettings, daily, monthly int64, now time.Time) econ.RewardBudget {
	return econ.RewardBudget{
		Daily: econ.RewardWindow{
			Started: daily,
			Max:     settings.DailyRewardCap,
			Renewal: nextDayStart(now),
		},
		Monthly: econ.RewardWindow{
			Started: monthly,
			Max:     settings.MonthlyRewardCap,
			Renewal: nextMonthStart(now),
		},
	}
}

// capsFilled reports whether either window is exhausted.
func capsFilled(settings econ.RewardSettings, daily, monthly int64) bool {
	return daily >= settings.DailyRewardCap || monthly >= settings.MonthlyRewardCap
}

// ------------------------------------------------------------------
// econ.Ads: user flow
// ------------------------------------------------------------------

// Pick implements econ.Ads.Pick.
func (a *Ads) Pick(ctx context.Context, userID, currency string) (econ.Ad, econ.RewardBudget, error) {
	s, err := a.settings()
	if err != nil {
		return econ.Ad{}, econ.RewardBudget{}, err
	}
	if currency == "" {
		currency = econ.CurrencyToken
	}
	if err := a.wallet.knownCurrency(currency); err != nil {
		return econ.Ad{}, econ.RewardBudget{}, err
	}

	now := a.now()
	daily, monthly, err := a.earnedInTx(a.store.DB, userID, currency, now)
	if err != nil {
		return econ.Ad{}, econ.RewardBudget{}, err
	}
	if capsFilled(s, daily, monthly) {
		return econ.Ad{}, budgetOf(s, daily, monthly, now), econ.ErrRewardLimitReached
	}

	ad, found, err := a.store.PickEligibleAd(a.store.DB, userID, utcDay(now), now)
	if err != nil {
		return econ.Ad{}, econ.RewardBudget{}, err
	}
	if !found {
		return econ.Ad{}, budgetOf(s, daily, monthly, now), econ.ErrAdNotAvailable
	}
	return adOf(ad), budgetOf(s, daily, monthly, now), nil
}

// StartView implements econ.Ads.StartView.
func (a *Ads) StartView(ctx context.Context, userID, adID string) (econ.AdViewSession, error) {
	ad, found, err := a.store.AdByID(a.store.DB, adID)
	if err != nil {
		return econ.AdViewSession{}, err
	}
	if !found || !ad.Active || ad.WatchReward <= 0 {
		return econ.AdViewSession{}, econ.ErrAdNotAvailable
	}

	now := a.now()
	if ad.StartsAt != nil && now.Before(*ad.StartsAt) {
		return econ.AdViewSession{}, econ.ErrAdNotAvailable
	}
	if ad.EndsAt != nil && now.After(*ad.EndsAt) {
		return econ.AdViewSession{}, econ.ErrAdNotAvailable
	}

	day := utcDay(now)
	s, err := a.settings()
	if err != nil {
		return econ.AdViewSession{}, err
	}
	daily, monthly, err := a.earnedInTx(a.store.DB, userID, econ.CurrencyToken, now)
	if err != nil {
		return econ.AdViewSession{}, err
	}
	if capsFilled(s, daily, monthly) {
		return econ.AdViewSession{}, econ.ErrRewardLimitReached
	}

	existing, found, err := a.store.OpenSessionToday(a.store.DB, userID, adID, day)
	if err != nil {
		return econ.AdViewSession{}, err
	}
	if found {
		// Today's session is burnt once used (rewarded) or expired; a new one
		// only becomes possible tomorrow, by the unique index.
		if existing.Status == econ.SessionRewarded || now.After(existing.ExpiresAt) {
			return econ.AdViewSession{}, econ.ErrSessionNotOpen
		}
		return sessionOf(existing), nil
	}

	v := model.AdViewSession{
		ID:              uuid.NewString(),
		AdID:            adID,
		UserID:          userID,
		RequiredSeconds: ad.WatchSeconds,
		Status:          econ.SessionStarted,
		ServerStartedAt: now,
		ExpiresAt:       sessionExpiry(now, ad.WatchSeconds),
		RewardMinor:     ad.WatchReward,
		RewardDate:      day,
	}
	if err := a.store.CreateSession(a.store.DB, v); err != nil {
		// A concurrent StartView for the same (user, ad, day) won: return
		// their session instead of surfacing a 500.
		if existing, found, eerr := a.store.OpenSessionToday(a.store.DB, userID, adID, day); eerr == nil && found {
			return sessionOf(existing), nil
		}
		return econ.AdViewSession{}, err
	}
	return sessionOf(v), nil
}

// ClaimView implements econ.Ads.ClaimView. The whole claim (session flip +
// ledger entry) is one transaction, and the movement carries the session id as
// its idempotency key so a network retry returns the original entry.
func (a *Ads) ClaimView(ctx context.Context, userID, sessionID string) (econ.Entry, error) {
	if sessionID == "" || userID == "" {
		return econ.Entry{}, econ.ErrSessionNotOpen
	}
	tx := a.store.DB.Begin()
	if tx.Error != nil {
		return econ.Entry{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	sess, found, err := a.store.SessionForUpdate(tx, sessionID)
	if err != nil {
		return econ.Entry{}, err
	}
	if !found || sess.UserID != userID || sess.Status != econ.SessionStarted {
		return econ.Entry{}, econ.ErrSessionNotOpen
	}

	now := a.now()
	if now.After(sess.ExpiresAt) {
		return econ.Entry{}, econ.ErrSessionNotOpen
	}
	if elapsed := now.Sub(sess.ServerStartedAt); elapsed < time.Duration(sess.RequiredSeconds)*time.Second {
		return econ.Entry{}, econ.ErrRewardTooEarly
	}

	s, err := settingsInTx(tx, now)
	if err != nil {
		return econ.Entry{}, err
	}
	daily, monthly, err := a.earnedInTx(tx, userID, econ.CurrencyToken, now)
	if err != nil {
		return econ.Entry{}, err
	}
	if capsFilled(s, daily, monthly) {
		return econ.Entry{}, econ.ErrRewardLimitReached
	}

	var entry econ.Entry
	if sess.RewardMinor > 0 {
		entry, err = a.wallet.moveOnTx(tx, econ.Movement{
			UserID:         userID,
			Currency:       econ.CurrencyToken,
			Amount:         sess.RewardMinor,
			Reason:         econ.ReasonAdWatchReward,
			RefType:        "ad_view_session",
			RefID:          sess.ID,
			IdempotencyKey: "ad_watch:" + sess.ID,
		}, +1)
		if err != nil {
			return econ.Entry{}, err
		}
	}

	if err := a.store.RewardSession(tx, sess.ID, entry.ID, sess.RewardMinor); err != nil {
		return econ.Entry{}, err
	}
	return entry, tx.Commit().Error
}

// RecordClick implements econ.Ads.RecordClick. The click is always recorded
// (it is an analytics event); the reward is paid only when the dials enable
// it, the ad carries a positive click reward, the user's first paid click for
// (ad, day) has room left in the shared caps.
func (a *Ads) RecordClick(ctx context.Context, userID, adID string) (string, *econ.Entry, error) {
	ad, found, err := a.store.AdByID(a.store.DB, adID)
	if err != nil {
		return "", nil, err
	}
	if !found || !ad.Active {
		return "", nil, econ.ErrAdNotAvailable
	}

	tx := a.store.DB.Begin()
	if tx.Error != nil {
		return "", nil, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	now := a.now()
	s, err := settingsInTx(tx, now)
	if err != nil {
		return "", nil, err
	}

	uid := ""
	if userID != "" {
		uid = userID
	}
	var day *time.Time
	if uid != "" {
		d := utcDay(now)
		day = &d
	}
	click := model.AdClick{
		ID:         uuid.NewString(),
		AdID:       adID,
		UserID:     nilIfEmpty(uid),
		Kind:       "click",
		RewardDate: day,
	}
	first, err := a.store.InsertClickUnlessDup(tx, click)
	if err != nil {
		return "", nil, err
	}

	var entry *econ.Entry
	if first && uid != "" && s.ClickRewardEnabled && ad.ClickReward > 0 {
		daily, monthly, eerr := a.earnedInTx(tx, uid, econ.CurrencyToken, now)
		if eerr != nil {
			return "", nil, eerr
		}
		if !capsFilled(s, daily, monthly) {
			e, err := a.wallet.moveOnTx(tx, econ.Movement{
				UserID:         uid,
				Currency:       econ.CurrencyToken,
				Amount:         ad.ClickReward,
				Reason:         econ.ReasonAdClickReward,
				RefType:        "ad",
				RefID:          adID,
				IdempotencyKey: "ad_click:" + uid + ":" + adID + ":" + utcDay(now).Format("2006-01-02"),
			}, +1)
			if err != nil {
				return "", nil, err
			}
			entry = &e
		}
	}

	if err := tx.Commit().Error; err != nil {
		return "", nil, err
	}
	return deref(ad.TargetURL), entry, nil
}

// Budget implements econ.Ads.Budget.
func (a *Ads) Budget(ctx context.Context, userID, currency string) (econ.RewardBudget, error) {
	s, err := a.settings()
	if err != nil {
		return econ.RewardBudget{}, err
	}
	if currency == "" {
		currency = econ.CurrencyToken
	}
	if err := a.wallet.knownCurrency(currency); err != nil {
		return econ.RewardBudget{}, err
	}
	now := a.now()
	daily, monthly, err := a.earnedInTx(a.store.DB, userID, currency, now)
	if err != nil {
		return econ.RewardBudget{}, err
	}
	return budgetOf(s, daily, monthly, now), nil
}

// ------------------------------------------------------------------
// econ.Ads: dials and admin surface
// ------------------------------------------------------------------

// Settings implements econ.Ads.Settings.
func (a *Ads) Settings(ctx context.Context) (econ.RewardSettings, error) {
	return a.settings()
}

func (a *Ads) settings() (econ.RewardSettings, error) {
	r, err := a.store.Settings(a.store.DB, a.now())
	if err != nil {
		return econ.RewardSettings{}, err
	}
	return settingsOf(r), nil
}

// UpdateSettings implements econ.Ads.UpdateSettings.
func (a *Ads) UpdateSettings(ctx context.Context, in econ.RewardSettings, updatedBy string) (econ.RewardSettings, error) {
	if in.SignupGrant < 0 || in.AdWatchReward < 0 || in.DailyRewardCap < 0 || in.MonthlyRewardCap < 0 {
		return econ.RewardSettings{}, core.ErrValidation
	}
	if in.MonthlyRewardCap > 0 && in.DailyRewardCap > in.MonthlyRewardCap {
		// A daily dial above the monthly one is almost always a mistake; the
		// monthly cap would shadow it entirely.
		return econ.RewardSettings{}, core.ErrValidation
	}
	if updatedBy == "" {
		return econ.RewardSettings{}, core.ErrValidation
	}

	tx := a.store.DB.Begin()
	if tx.Error != nil {
		return econ.RewardSettings{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	// Ensure the row exists before updating it.
	if _, err := settingsInTx(tx, a.now()); err != nil {
		return econ.RewardSettings{}, err
	}
	row := model.RewardSettings{
		ID:                 1,
		SignupGrant:        in.SignupGrant,
		AdWatchReward:      in.AdWatchReward,
		DailyRewardCap:     in.DailyRewardCap,
		MonthlyRewardCap:   in.MonthlyRewardCap,
		ClickRewardEnabled: in.ClickRewardEnabled,
		UpdatedBy:          nilIfEmpty(updatedBy),
	}
	if err := a.store.SaveSettings(tx, row, a.now()); err != nil {
		return econ.RewardSettings{}, err
	}
	if err := tx.Commit().Error; err != nil {
		return econ.RewardSettings{}, err
	}
	out, err := a.settings()
	if err != nil {
		return econ.RewardSettings{}, err
	}
	return out, nil
}

// CreateAd implements econ.Ads.CreateAd. A zero WatchReward means "use the
// admin dial" (reward_settings.ad_watch_reward): the amount is always a server
// decision, never a client one (L2.md 3.4).
func (a *Ads) CreateAd(ctx context.Context, in econ.AdInput) (econ.Ad, error) {
	if err := validateAdInput(in); err != nil {
		return econ.Ad{}, err
	}
	if in.WatchReward == 0 {
		s, err := a.settings()
		if err != nil {
			return econ.Ad{}, err
		}
		in.WatchReward = s.AdWatchReward
	}
	m, err := adRow(in)
	if err != nil {
		return econ.Ad{}, err
	}
	m.ID = uuid.NewString()
	m.CreatedAt = a.now()
	m.UpdatedAt = a.now()
	if err := a.store.InsertAd(a.store.DB, m); err != nil {
		return econ.Ad{}, err
	}
	return adOf(m), nil
}

// UpdateAd implements econ.Ads.UpdateAd.
func (a *Ads) UpdateAd(ctx context.Context, id string, in econ.AdInput) (econ.Ad, error) {
	if err := validateAdInput(in); err != nil {
		return econ.Ad{}, err
	}
	existing, found, err := a.store.AdByID(a.store.DB, id)
	if err != nil {
		return econ.Ad{}, err
	}
	if !found {
		return econ.Ad{}, core.ErrNotFound
	}
	m, err := adRow(in)
	if err != nil {
		return econ.Ad{}, err
	}
	m.ID = existing.ID
	m.CreatedAt = existing.CreatedAt
	m.UpdatedAt = a.now()
	if err := a.store.UpdateAd(a.store.DB, m); err != nil {
		return econ.Ad{}, err
	}
	return adOf(m), nil
}

// DeleteAd implements econ.Ads.DeleteAd.
func (a *Ads) DeleteAd(ctx context.Context, id string) error {
	// Existence first: hard-delete reports nothing about missing rows, and
	// 404 is what an admin sees when the id is stale.
	_, found, err := a.store.AdByID(a.store.DB, id)
	if err != nil {
		return err
	}
	if !found {
		return core.ErrNotFound
	}
	return a.store.DeleteAd(a.store.DB, id)
}

// ListAds implements econ.Ads.ListAds.
func (a *Ads) ListAds(ctx context.Context, limit, offset int, activeOnly bool) ([]econ.Ad, int, error) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > hardPageLimit {
		limit = hardPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	rows, total, err := a.store.ListAds(a.store.DB, limit, offset, activeOnly)
	if err != nil {
		return nil, 0, err
	}
	out := make([]econ.Ad, 0, len(rows))
	for _, r := range rows {
		out = append(out, adOf(r))
	}
	return out, total, nil
}

// AdStats implements econ.Ads.AdStats.
func (a *Ads) AdStats(ctx context.Context, id string) (econ.AdStats, error) {
	sessions, rewarded, rewardedSum, clicks, clickRewards, err := a.store.AdStats(a.store.DB, id)
	if err != nil {
		return econ.AdStats{}, err
	}
	return econ.AdStats{
		AdID:         id,
		Sessions:     sessions,
		Rewarded:     rewarded,
		RewardedSum:  rewardedSum,
		Clicks:       clicks,
		ClickRewards: clickRewards,
	}, nil
}

// ------------------------------------------------------------------
// mapping + validation helpers
// ------------------------------------------------------------------

func adOf(m model.Ad) econ.Ad {
	return econ.Ad{
		ID:           m.ID,
		Code:         m.Code,
		Title:        m.Title,
		MediaType:    m.MediaType,
		BodyText:     deref(m.BodyText),
		MediaURL:     deref(m.MediaURL),
		Content:      deref(m.Content),
		TargetURL:    deref(m.TargetURL),
		WatchSeconds: m.WatchSeconds,
		WatchReward:  m.WatchReward,
		ClickReward:  m.ClickReward,
		StartsAt:     m.StartsAt,
		EndsAt:       m.EndsAt,
		Active:       m.Active,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func sessionOf(v model.AdViewSession) econ.AdViewSession {
	return econ.AdViewSession{
		ID:              v.ID,
		AdID:            v.AdID,
		UserID:          v.UserID,
		RequiredSeconds: v.RequiredSeconds,
		Status:          v.Status,
		ServerStartedAt: v.ServerStartedAt,
		ExpiresAt:       v.ExpiresAt,
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// adRow builds the model row from the admin payload.
func adRow(in econ.AdInput) (model.Ad, error) {
	m := model.Ad{
		Code:         strings.TrimSpace(in.Code),
		Title:        strings.TrimSpace(in.Title),
		MediaType:    in.MediaType,
		WatchSeconds: in.WatchSeconds,
		WatchReward:  in.WatchReward,
		ClickReward:  in.ClickReward,
		StartsAt:     in.StartsAt,
		EndsAt:       in.EndsAt,
		Active:       in.Active,
	}
	m.BodyText = nilIfEmpty(strings.TrimSpace(in.BodyText))
	m.MediaURL = nilIfEmpty(strings.TrimSpace(in.MediaURL))
	m.Content = nilIfEmpty(strings.TrimSpace(in.Content))
	m.TargetURL = nilIfEmpty(strings.TrimSpace(in.TargetURL))
	return m, nil
}

// validateAdInput enforces the same rules the schema's CHECK constraints do,
// plus the https-only affiliate policy, so an admin gets a 4xx instead of an
// ugly database error.
func validateAdInput(in econ.AdInput) error {
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Title) == "" {
		return econ.ErrAdInvalid
	}
	switch in.MediaType {
	case "video", "image", "text":
	default:
		return econ.ErrAdInvalid
	}
	if in.MediaType == "text" {
		if strings.TrimSpace(in.BodyText) == "" || strings.TrimSpace(in.MediaURL) != "" {
			return econ.ErrAdInvalid
		}
	} else if strings.TrimSpace(in.MediaURL) == "" {
		return econ.ErrAdInvalid
	}
	if in.TargetURL != "" && !strings.HasPrefix(strings.TrimSpace(in.TargetURL), "https://") {
		// Open-redirect defence: affiliate links are https-only.
		return econ.ErrAdInvalid
	}
	if in.WatchSeconds <= 0 {
		return econ.ErrAdInvalid
	}
	if in.WatchReward < 0 || in.ClickReward < 0 {
		return econ.ErrAdInvalid
	}
	if in.StartsAt != nil && in.EndsAt != nil && in.EndsAt.Before(*in.StartsAt) {
		return econ.ErrAdInvalid
	}
	return nil
}
