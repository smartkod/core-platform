package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
)

// newAds builds the rewarded-ads service over a fresh user.
func newAds(t *testing.T, db *gorm.DB, clock *fakeClock) (*service.Ads, *service.Wallet, string) {
	t.Helper()
	store := repository.NewStore(db)
	w := service.NewWallet(store, nil, clock)
	return service.NewAds(w, store, clock.Now), w, uuid.NewString()
}

// createAd is the admin path, which also exercises payload validation. The
// ad is removed after the test so shared databases do not accumulate eligible
// creatives that would skew a later Pick.
func createAd(t *testing.T, db *gorm.DB, ads *service.Ads, code string, watch int, reward, click int64) econ.Ad {
	t.Helper()
	ad, err := ads.CreateAd(context.Background(), econ.AdInput{
		Code:         code,
		Title:        "Test " + code,
		MediaType:    "video",
		MediaURL:     "https://cdn.example.com/" + code + ".mp4",
		TargetURL:    "https://affiliate.example.com/" + code,
		WatchSeconds: watch,
		WatchReward:  reward,
		ClickReward:  click,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("create ad %s: %v", code, err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM ads WHERE id = ?`, ad.ID) })
	return ad
}

func TestAdsWatchFlow(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	ads, wallet, userID := newAds(t, db, clock)
	ctx := context.Background()

	createAd(t, db, ads, "flow-"+userID[:8], 30, 1, 0)

	ad, budget, err := ads.Pick(ctx, userID, econ.CurrencyToken)
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	if budget.Daily.Max != 10 {
		t.Fatalf("budget max: %d", budget.Daily.Max)
	}

	sess, err := ads.StartView(ctx, userID, ad.ID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if sess.RequiredSeconds != ad.WatchSeconds {
		t.Fatalf("required seconds: got %d want %d", sess.RequiredSeconds, ad.WatchSeconds)
	}

	// Before the server clock says the duration elapsed: refused.
	if _, err := ads.ClaimView(ctx, userID, sess.ID); !errors.Is(err, econ.ErrRewardTooEarly) {
		t.Fatalf("early claim: want ErrRewardTooEarly, got %v", err)
	}

	clock.Advance(time.Duration(ad.WatchSeconds)*time.Second + time.Second)
	entry, err := ads.ClaimView(ctx, userID, sess.ID)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if entry.Delta != 1 || entry.Reason != econ.ReasonAdWatchReward {
		t.Fatalf("unexpected reward entry: %+v", entry)
	}

	// Second claim on the same session is impossible.
	if _, err := ads.ClaimView(ctx, userID, sess.ID); !errors.Is(err, econ.ErrSessionNotOpen) {
		t.Fatalf("double claim: want ErrSessionNotOpen, got %v", err)
	}
	// A second session for the same (user, ad, day) is structurally refused.
	if _, err := ads.StartView(ctx, userID, ad.ID); !errors.Is(err, econ.ErrSessionNotOpen) {
		t.Fatalf("second session same day: want ErrSessionNotOpen, got %v", err)
	}

	bal, _ := wallet.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 1 {
		t.Fatalf("balance after one watch: %d", bal.Balance)
	}
}

func TestAdsDailyCap(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 3, 10, false)
	ads, wallet, userID := newAds(t, db, clock)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		ad := createAd(t, db, ads, fmt.Sprintf("cap-%s-%d", userID[:6], i), 5, 1, 0)
		sess, err := ads.StartView(ctx, userID, ad.ID)
		if err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
		clock.Advance(6 * time.Second)
		if _, err := ads.ClaimView(ctx, userID, sess.ID); err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
	}

	// Fourth ad exists, but the daily cap of 3 is full.
	fourth := createAd(t, db, ads, "cap-"+userID[:6]+"-fourth", 5, 1, 0)
	if _, err := ads.StartView(ctx, userID, fourth.ID); !errors.Is(err, econ.ErrRewardLimitReached) {
		t.Fatalf("start past cap: want ErrRewardLimitReached, got %v", err)
	}
	if _, _, err := ads.Pick(ctx, userID, econ.CurrencyToken); !errors.Is(err, econ.ErrRewardLimitReached) {
		t.Fatalf("pick past cap: want ErrRewardLimitReached, got %v", err)
	}

	bal, _ := wallet.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 3 {
		t.Fatalf("balance after cap: %d", bal.Balance)
	}

	// Tomorrow the window resets.
	clock.Advance(24 * time.Hour)
	budget, err := ads.Budget(ctx, userID, econ.CurrencyToken)
	if err != nil {
		t.Fatal(err)
	}
	if budget.Daily.Started != 0 {
		t.Fatalf("daily window did not reset: started=%d", budget.Daily.Started)
	}
	if budget.Monthly.Started != 3 {
		t.Fatalf("monthly total should persist: %d", budget.Monthly.Started)
	}
}

func TestAdsClickRewardOncePerDay(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	ads, wallet, userID := newAds(t, db, clock)
	ctx := context.Background()

	ad := createAd(t, db, ads, "click-"+userID[:8], 5, 1, 2)

	target, entry, err := ads.RecordClick(ctx, userID, ad.ID)
	if err != nil {
		t.Fatalf("click: %v", err)
	}
	if target != "https://affiliate.example.com/"+ad.Code {
		t.Fatalf("affiliate target: %q", target)
	}
	if entry == nil || entry.Delta != 2 {
		t.Fatalf("click reward: %+v", entry)
	}

	// Same day, second click: logged but not paid again.
	_, second, err := ads.RecordClick(ctx, userID, ad.ID)
	if err != nil {
		t.Fatalf("second click: %v", err)
	}
	if second != nil {
		t.Fatalf("click rewarded twice: %+v", second)
	}

	bal, _ := wallet.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 2 {
		t.Fatalf("balance after clicks: %d", bal.Balance)
	}
}

func TestAdsClickDisabledPaysNothing(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, false)
	ads, wallet, userID := newAds(t, db, clock)
	ctx := context.Background()

	ad := createAd(t, db, ads, "noclick-"+userID[:8], 5, 1, 2)
	_, entry, err := ads.RecordClick(ctx, userID, ad.ID)
	if err != nil {
		t.Fatal(err)
	}
	if entry != nil {
		t.Fatalf("click reward paid while disabled: %+v", entry)
	}
	bal, _ := wallet.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 0 {
		t.Fatalf("balance: %d", bal.Balance)
	}
}

func TestAdInputValidation(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	ads, _, _ := newAds(t, db, clock)
	ctx := context.Background()

	bad := []econ.AdInput{
		{Code: "x", Title: "t", MediaType: "video", MediaURL: "https://a/b.mp4", WatchSeconds: 1, TargetURL: "http://plain.example.com"},
		{Code: "x", Title: "t", MediaType: "audio", MediaURL: "https://a/b.mp3", WatchSeconds: 1},
		{Code: "x", Title: "t", MediaType: "text", WatchSeconds: 1},
		{Code: "x", Title: "t", MediaType: "video", WatchSeconds: 1},
		{Code: "", Title: "t", MediaType: "video", MediaURL: "https://a/b.mp4", WatchSeconds: 1},
		{Code: "x", Title: "t", MediaType: "video", MediaURL: "https://a/b.mp4", WatchSeconds: 0},
	}
	for i, in := range bad {
		if _, err := ads.CreateAd(ctx, in); !errors.Is(err, econ.ErrAdInvalid) {
			t.Fatalf("case %d: want ErrAdInvalid, got %v", i, err)
		}
	}
}

func TestUpdateSettingsRecordsAudit(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 10, 1, 3, 10, true)
	ads, _, _ := newAds(t, db, clock)
	ctx := context.Background()

	admin := uuid.NewString()
	out, err := ads.UpdateSettings(ctx, econ.RewardSettings{
		SignupGrant: 20, AdWatchReward: 2, DailyRewardCap: 5,
		MonthlyRewardCap: 20, ClickRewardEnabled: false,
	}, admin)
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if out.DailyRewardCap != 5 || out.UpdatedBy != admin || out.ClickRewardEnabled {
		t.Fatalf("settings not applied: %+v", out)
	}

	// A daily cap above the monthly cap is rejected as a dial mistake.
	if _, err := ads.UpdateSettings(ctx, econ.RewardSettings{
		SignupGrant: 5, AdWatchReward: 1, DailyRewardCap: 50,
		MonthlyRewardCap: 10, ClickRewardEnabled: true,
	}, admin); !errors.Is(err, core.ErrValidation) {
		t.Fatalf("want validation error, got %v", err)
	}
}

func TestSettingsPersistAppliedReward(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	ads, wallet, userID := newAds(t, db, clock)
	ctx := context.Background()

	if _, err := ads.UpdateSettings(ctx, econ.RewardSettings{
		SignupGrant: 0, AdWatchReward: 4, DailyRewardCap: 10,
		MonthlyRewardCap: 100, ClickRewardEnabled: true,
	}, uuid.NewString()); err != nil {
		t.Fatal(err)
	}

	ad := createAd(t, db, ads, "dial-"+userID[:8], 5, 4, 0)
	sess, err := ads.StartView(ctx, userID, ad.ID)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(6 * time.Second)
	entry, err := ads.ClaimView(ctx, userID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Delta != 4 {
		t.Fatalf("session reward should snapshot the ad's reward: %d", entry.Delta)
	}
	bal, _ := wallet.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 4 {
		t.Fatalf("balance: %d", bal.Balance)
	}
}
