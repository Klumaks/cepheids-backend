package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"

	"cepheids-backend/internal/app/auth"
	"cepheids-backend/internal/app/ds"
	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCepheidLab4Integration(t *testing.T) {
	databaseURL := os.Getenv("CEPHEIDS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CEPHEIDS_TEST_DATABASE_URL to an isolated PostgreSQL database containing cepheids_lab4_test in its name")
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var databaseName string
	if err := db.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(databaseName, "cepheids_lab4_test") {
		t.Fatal("integration tests require an isolated cepheids_lab4_test database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&ds.User{}, &ds.SpectralClass{}, &ds.Like{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_spectral_class_observer_draft ON spectral_classes (creator_id) WHERE status = 'draft'").Error; err != nil {
		t.Fatal(err)
	}
	repo, err := repository.New(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	media, err := pkg.NewMinioClient("localhost:9000", "minioadmin", "minioadmin", "cepheids")
	if err != nil {
		t.Fatal(err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { redisClient.Close() })
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	tokens, _ := auth.NewCepheidTokens(strings.Repeat("integration-secret-", 3), time.Hour, auth.RedisCepheidBlacklist{Client: redisClient})
	h := NewHandler(repo, media, tokens)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h.RegisterHandler(router)
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	credentials := func(login string) string {
		return fmt.Sprintf("{\"login\":%q,\"password\":\"cepheid-test-password\"}", login)
	}
	send := func(method, path, token string, body io.Reader, contentType string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, body)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("%s %s: got %d want %d; body=%s", method, path, out.Code, want, out.Body.String())
		}
		return out.Body.Bytes()
	}
	call := func(method, path, token, body string, want int) []byte {
		return send(method, path, token, strings.NewReader(body), "application/json", want)
	}
	register := func(login string) ds.User {
		var observer ds.User
		json.Unmarshal(call("POST", "/api/users/register", "", credentials(login), 201), &observer)
		var stored ds.User
		if err := db.First(&stored, observer.ID).Error; err != nil {
			t.Fatal(err)
		}
		if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("cepheid-test-password")) != nil {
			t.Fatal("password not hashed")
		}
		return observer
	}
	firstLogin, secondLogin := "first_"+suffix, "second_"+suffix
	first := register(firstLogin)
	second := register(secondLogin)
	call("POST", "/api/users/register", "", credentials(firstLogin), 409)
	call("POST", "/api/users/register", "", "{\"login\":\"\",\"password\":\"x\"}", 400)
	call("POST", "/api/users/login", "", "{\"login\":\"missing\",\"password\":\"wrong\"}", 401)
	login := func(name string) string {
		var response CepheidLoginResponse
		json.Unmarshal(call("POST", "/api/users/login", "", credentials(name), 200), &response)
		if response.AccessToken == "" || response.ExpiresIn != 3600 || response.TokenType != "Bearer" {
			t.Fatal("invalid login response")
		}
		return response.AccessToken
	}
	firstToken, secondToken := login(firstLogin), login(secondLogin)
	// Use a real unreachable Redis address without stopping the project's Redis.
	failedRedis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 100 * time.Millisecond, MaxRetries: -1})
	defer failedRedis.Close()
	failedTokens, _ := auth.NewCepheidTokens(strings.Repeat("integration-secret-", 3), time.Hour, auth.RedisCepheidBlacklist{Client: failedRedis})
	failedRouter := gin.New()
	NewHandler(repo, media, failedTokens).RegisterHandler(failedRouter)
	normalRouter := router
	router = failedRouter
	call("GET", "/api/cepheids/draft", firstToken, "", 503)
	call("POST", "/api/users/logout", firstToken, "", 503)
	call("GET", "/api/cepheids", firstToken, "", 200)
	router = normalRouter
	publication := "{\"description\":\"Цефеиды класса F: калибровка\",\"pl_slope\":-2.76,\"pl_intercept\":-1.4}"
	call("PUT", "/api/cepheids/1/publish", "", publication, 401)
	call("GET", "/api/cepheids/draft", "broken", "", 401)
	call("GET", "/api/cepheids/draft", firstToken, "", 404)
	call("GET", "/api/cepheids", "broken", "", 200)
	create := func(token string, withMedia bool, want int) ds.SpectralClass {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		form.WriteField("name", "Cepheid F calibration")
		form.WriteField("creator_id", fmt.Sprint(second.ID))
		if withMedia {
			for _, kind := range []string{"image", "video"} {
				extension := ".jpg"
				if kind == "video" {
					extension = ".mp4"
				}
				data, err := os.ReadFile("../../../resources/img/default" + extension)
				if err != nil {
					t.Fatal(err)
				}
				part, err := form.CreateFormFile(kind, "default"+extension)
				if err != nil {
					t.Fatal(err)
				}
				part.Write(data)
			}
		}
		form.Close()
		var c ds.SpectralClass
		json.Unmarshal(send("POST", "/api/cepheids", token, &body, form.FormDataContentType(), want), &c)
		return c
	}
	c := create(firstToken, true, 201)
	if c.CreatorID != first.ID || c.ImageKey == "" || c.VideoKey == "" {
		t.Fatal("creator or media incorrect")
	}
	t.Cleanup(func() { media.RemoveFile(c.ImageKey); media.RemoveFile(c.VideoKey) })
	call("GET", "/api/cepheids/draft", firstToken, "", 200)
	create(firstToken, false, 409)
	other := create(secondToken, false, 201)
	if other.CreatorID != second.ID {
		t.Fatal("wrong second creator")
	}
	call("PUT", fmt.Sprintf("/api/cepheids/%d/publish", c.ID), secondToken, publication, 403)
	call("DELETE", fmt.Sprintf("/api/cepheids/%d", c.ID), secondToken, "", 403)
	call("PUT", fmt.Sprintf("/api/cepheids/%d/publish", c.ID), firstToken, "{}", 400)
	published := call("PUT", fmt.Sprintf("/api/cepheids/%d/publish", c.ID), firstToken, publication, 200)
	var updated ds.SpectralClass
	json.Unmarshal(published, &updated)
	if updated.PlSlope == nil || *updated.PlSlope != -2.76 || updated.IsMine != 1 {
		t.Fatal("publication fields not saved")
	}
	call("PUT", fmt.Sprintf("/api/cepheids/%d/publish", c.ID), firstToken, publication, 404)
	call("GET", "/api/cepheids?min_slope=-3&max_slope=-2", firstToken, "", 200)
	var guest []ds.SpectralClass
	json.Unmarshal(call("GET", "/api/cepheids", "broken", "", 200), &guest)
	for _, entry := range guest {
		if entry.IsMine != 0 || entry.IsLiked != 0 {
			t.Fatal("guest got personal flags")
		}
	}
	call("GET", "/api/cepheids?min_slope=NaN", firstToken, "", 400)
	call("GET", "/api/cepheids/feed", "", "", 200)
	call("GET", fmt.Sprintf("/api/cepheids/feed/%d?next=true", c.ID), "", "", 200)
	likePath := fmt.Sprintf("/api/cepheids/%d/like", c.ID)
	call("POST", likePath, firstToken, "{\"like\":2}", 400)
	call("POST", likePath, firstToken, "{}", 400)
	call("POST", likePath, firstToken, "{\"like\":1}", 200)
	call("POST", likePath, firstToken, "{\"like\":1}", 200)
	var liked ds.SpectralClass
	json.Unmarshal(call("GET", fmt.Sprintf("/api/cepheids/feed/%d", c.ID), firstToken, "", 200), &liked)
	if liked.IsLiked != 1 || liked.LikesCount != 1 {
		t.Fatal("like not idempotent")
	}
	call("POST", likePath, firstToken, "{\"like\":0}", 200)
	claims, _, _ := tokens.Validate(context.Background(), firstToken)
	call("POST", "/api/users/logout", firstToken, "", 200)
	blacklistKey := "cepheids:jwt:blacklist:" + claims.ID
	ttl, err := redisClient.TTL(context.Background(), blacklistKey).Result()
	if err != nil || ttl <= 0 || ttl > time.Hour {
		t.Fatalf("blacklist TTL %v: %v", ttl, err)
	}
	t.Cleanup(func() { redisClient.Del(context.Background(), blacklistKey) })
	call("GET", "/api/cepheids/draft", firstToken, "", 401)
	call("GET", "/api/cepheids", firstToken, "", 200)
	call("POST", "/api/users/logout", firstToken, "", 401)
	firstToken = login(firstLogin)
	call("DELETE", fmt.Sprintf("/api/cepheids/%d", c.ID), firstToken, "", 200)
	call("GET", fmt.Sprintf("/api/cepheids/feed/%d", c.ID), "", "", 404)
	call("POST", likePath, firstToken, "{\"like\":1}", 404)
	call("DELETE", fmt.Sprintf("/api/cepheids/%d", c.ID), firstToken, "", 404)
	call("GET", "/api/cepheids/feed", "", "", 404)

	// Competing draft creations must be resolved by the PostgreSQL unique index.
	third := register("third_" + suffix)
	var group sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := repo.CreateCepheid(third.ID, "Concurrent cepheid", "", "")
			results <- err
		}()
	}
	group.Wait()
	close(results)
	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent drafts: %d success, %d failures", successes, failures)
	}
	t.Log("Real PostgreSQL/MinIO/Redis lifecycle, two observers, upload, blacklist TTL, ownership, likes and concurrent drafts passed")

}
