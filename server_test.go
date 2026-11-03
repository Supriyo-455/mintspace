package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func createIsolatedTestServer(t *testing.T) *Server {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	blobDir := filepath.Join(tempDir, "blobs")

	server := new(Server)
	server.storage = NewSqliteStorage(dbPath)
	localStore, err := NewLocalCASStorage(blobDir)
	if err != nil {
		t.Fatalf("failed to init test CAS: %v", err)
	}
	server.blobStorage = NewDistributedCASStorage(localStore, nil)
	server.authLimiter = NewIPRateLimiter(100, 100)
	server.writeLimiter = NewIPRateLimiter(100, 100)
	server.RegisterRoutes()

	if err := server.storage.Connect(); err != nil {
		t.Fatalf("failed to connect test sqlite: %v", err)
	}
	server.seedDemoPaperIfEmpty()

	t.Cleanup(func() {
		server.storage.Disconnect()
	})

	return server
}

func TestCommentDeletion(t *testing.T) {
	server := createIsolatedTestServer(t)

	// Create a blog under admin
	blog := &Blog{
		AuthorEmail: "admin@mintspace.io",
		Title:       "Test Paper",
		ContentHash: "dummyhash",
		DateCreated: "2026-01-01",
	}
	blogId, err := server.storage.CreateBlog(blog)
	if err != nil {
		t.Fatalf("failed to create blog: %v", err)
	}

	// Create comment
	comment := &Comment{
		BlogId:      blogId,
		UserEmail:   "admin@mintspace.io",
		AuthorName:  "Admin",
		Content:     "Temporary comment",
		DateCreated: "2026-01-01 10:00",
	}
	commentId, err := server.storage.CreateComment(comment)
	if err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}

	// Verify it exists
	c, err := server.storage.GetCommentById(commentId)
	if err != nil || c == nil {
		t.Fatalf("expected comment to exist, got: %v", err)
	}

	// Delete it
	if err := server.storage.DeleteComment(commentId); err != nil {
		t.Fatalf("failed to delete comment: %v", err)
	}

	// Verify it's gone
	comments, _ := server.storage.GetCommentsByBlogId(blogId, "")
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments after delete, got %d", len(comments))
	}
}

func TestArchiveAndPaperView(t *testing.T) {
	server := createIsolatedTestServer(t)

	// 1. Test GET /blog
	req := httptest.NewRequest("GET", "/blog", nil)
	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /blog returned %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Content-Addressable Storage") {
		t.Fatalf("expected seeded demo paper in /blog response")
	}

	// 2. Test GET /blog/1
	reqDetail := httptest.NewRequest("GET", "/blog/1", nil)
	rrDetail := httptest.NewRecorder()
	server.mux.ServeHTTP(rrDetail, reqDetail)

	if rrDetail.Code != http.StatusOK {
		t.Fatalf("GET /blog/1 returned %d: %s", rrDetail.Code, rrDetail.Body.String())
	}
	if !strings.Contains(rrDetail.Body.String(), "ABSTRACT") {
		t.Fatalf("expected abstract section in paper view")
	}

	// 3. Test CAS Raw Blob retrieval
	blogs, err := server.storage.GetAllBlogs("", true)
	if err != nil || len(blogs) == 0 {
		t.Fatalf("failed to get blogs: %v", err)
	}
	contentHash := blogs[0].ContentHash

	reqBlob := httptest.NewRequest("GET", "/blob/"+contentHash, nil)
	rrBlob := httptest.NewRecorder()
	server.mux.ServeHTTP(rrBlob, reqBlob)

	if rrBlob.Code != http.StatusOK {
		t.Fatalf("GET /blob/%s returned %d", contentHash, rrBlob.Code)
	}
	if !strings.Contains(rrBlob.Body.String(), "## Abstract") {
		t.Fatalf("expected raw markdown content from blob endpoint")
	}
}

func TestWriteBlogAndCASVerification(t *testing.T) {
	server := createIsolatedTestServer(t)

	// Register author user
	_ = server.storage.CreateUser(&User{
		Name:              "Dr. Test Author",
		Email:             "author@test.edu",
		EncryptedPassword: "hash",
		Admin:             false,
		DateCreated:       "2026-01-01",
	})

	authorToken, err := createJWT("author@test.edu", "Dr. Test Author", false)
	if err != nil {
		t.Fatalf("jwt creation failed: %v", err)
	}

	form := url.Values{}
	form.Set("title", "Quantum Computing in Hashed Networks")
	form.Set("abstract", "An empirical study of qubits in CAS distributed systems.")
	form.Set("content", "# Quantum CAS\n\nExperimental proof of $\\psi = \\alpha |0\\rangle + \\beta |1\\rangle$.")
	form.Set("is_private", "true")

	req := httptest.NewRequest("POST", "/write", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "Auth", Value: authorToken})

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect after paper submission, got %d: %s", rr.Code, rr.Body.String())
	}

	userBlogs, err := server.storage.GetBlogsByAuthor("author@test.edu")
	if err != nil || len(userBlogs) == 0 {
		t.Fatalf("expected author paper to exist in DB")
	}
	newPaper := userBlogs[0]
	if !newPaper.IsPrivate {
		t.Fatalf("expected paper to be marked private")
	}

	blobBytes, err := server.blobStorage.Get(newPaper.ContentHash)
	if err != nil {
		t.Fatalf("CAS blob not found for hash %s", newPaper.ContentHash)
	}
	if !strings.Contains(string(blobBytes), "Quantum CAS") {
		t.Fatalf("content in CAS does not match expected markdown")
	}
}

func TestLikesAndComments(t *testing.T) {
	server := createIsolatedTestServer(t)

	_ = server.storage.CreateUser(&User{
		Name:              "Peer Reviewer",
		Email:             "reviewer@acm.org",
		EncryptedPassword: "hash",
		Admin:             false,
		DateCreated:       "2026-01-01",
	})
	token, _ := createJWT("reviewer@acm.org", "Peer Reviewer", false)

	// 1. Test Like
	reqLike := httptest.NewRequest("POST", "/blog/1/like", nil)
	reqLike.AddCookie(&http.Cookie{Name: "Auth", Value: token})
	rrLike := httptest.NewRecorder()
	server.mux.ServeHTTP(rrLike, reqLike)

	if rrLike.Code != http.StatusOK {
		t.Fatalf("like returned status %d", rrLike.Code)
	}
	var likeResp map[string]interface{}
	_ = json.Unmarshal(rrLike.Body.Bytes(), &likeResp)
	if likeResp["hasLiked"] != true {
		t.Fatalf("expected hasLiked=true, got %v", likeResp["hasLiked"])
	}

	// 2. Test Comment
	commentForm := url.Values{}
	commentForm.Set("comment", "Rigorous methodology and reproducible results.")
	reqComment := httptest.NewRequest("POST", "/blog/1/comment", strings.NewReader(commentForm.Encode()))
	reqComment.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqComment.AddCookie(&http.Cookie{Name: "Auth", Value: token})

	rrComment := httptest.NewRecorder()
	server.mux.ServeHTTP(rrComment, reqComment)

	if rrComment.Code != http.StatusSeeOther {
		t.Fatalf("comment returned status %d", rrComment.Code)
	}

	comments, err := server.storage.GetCommentsByBlogId(1, "reviewer@acm.org")
	if err != nil || len(comments) < 1 {
		t.Fatalf("expected at least 1 root comment on paper 1, got %d", len(comments))
	}

	parentCommentId := comments[0].Id

	// 3. Test Reply to Comment
	replyForm := url.Values{}
	replyForm.Set("comment", "I agree with your perspective on data structures.")
	replyForm.Set("parent_id", fmt.Sprintf("%d", parentCommentId))
	reqReply := httptest.NewRequest("POST", "/blog/1/comment", strings.NewReader(replyForm.Encode()))
	reqReply.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqReply.AddCookie(&http.Cookie{Name: "Auth", Value: token})

	rrReply := httptest.NewRecorder()
	server.mux.ServeHTTP(rrReply, reqReply)
	if rrReply.Code != http.StatusSeeOther {
		t.Fatalf("reply returned status %d", rrReply.Code)
	}

	// 4. Test Reply to Reply (Nested Threading)
	updatedComments, err := server.storage.GetCommentsByBlogId(1, "reviewer@acm.org")
	if err != nil {
		t.Fatalf("failed to fetch updated comments: %v", err)
	}

	var foundParent *Comment
	for i := range updatedComments {
		if updatedComments[i].Id == parentCommentId {
			foundParent = &updatedComments[i]
			break
		}
	}
	if foundParent == nil || len(foundParent.Replies) == 0 {
		t.Fatalf("expected reply under parent comment")
	}

	replyId := foundParent.Replies[0].Id

	// Reply to reply
	nestedReplyForm := url.Values{}
	nestedReplyForm.Set("comment", "Can you elaborate on the cache coherency aspects?")
	nestedReplyForm.Set("parent_id", fmt.Sprintf("%d", replyId))
	reqNested := httptest.NewRequest("POST", "/blog/1/comment", strings.NewReader(nestedReplyForm.Encode()))
	reqNested.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqNested.AddCookie(&http.Cookie{Name: "Auth", Value: token})

	rrNested := httptest.NewRecorder()
	server.mux.ServeHTTP(rrNested, reqNested)
	if rrNested.Code != http.StatusSeeOther {
		t.Fatalf("nested reply returned status %d", rrNested.Code)
	}

	// 5. Test Liking a Comment
	reqLikeComment := httptest.NewRequest("POST", fmt.Sprintf("/comment/%d/like", replyId), nil)
	reqLikeComment.AddCookie(&http.Cookie{Name: "Auth", Value: token})
	rrLikeComment := httptest.NewRecorder()
	server.mux.ServeHTTP(rrLikeComment, reqLikeComment)

	if rrLikeComment.Code != http.StatusOK {
		t.Fatalf("comment like returned status %d", rrLikeComment.Code)
	}
	var commentLikeResp map[string]interface{}
	_ = json.Unmarshal(rrLikeComment.Body.Bytes(), &commentLikeResp)
	if commentLikeResp["hasLiked"] != true {
		t.Fatalf("expected hasLiked=true on comment like, got %v", commentLikeResp["hasLiked"])
	}

	// 6. Verify nested comment structure and like reflection
	finalComments, err := server.storage.GetCommentsByBlogId(1, "reviewer@acm.org")
	if err != nil {
		t.Fatalf("failed to get final comments: %v", err)
	}
	var root *Comment
	for i := range finalComments {
		if finalComments[i].Id == parentCommentId {
			root = &finalComments[i]
			break
		}
	}
	if root == nil || len(root.Replies) == 0 {
		t.Fatalf("root comment missing replies")
	}
	firstReply := root.Replies[0]
	if len(firstReply.Replies) == 0 {
		t.Fatalf("expected reply-to-reply nested under first reply")
	}
	if !firstReply.UserHasLiked || firstReply.LikesCount != 1 {
		t.Fatalf("expected firstReply to have 1 like and UserHasLiked=true, got count=%d, liked=%v", firstReply.LikesCount, firstReply.UserHasLiked)
	}
}

func TestPrivateBlogRestrictions(t *testing.T) {
	server := createIsolatedTestServer(t)

	_ = server.storage.CreateUser(&User{Name: "Private Author", Email: "priv@test.edu", EncryptedPassword: "x", DateCreated: "2026-01-01"})
	hash, _ := server.blobStorage.Save([]byte("# Confidential Paper\nSecret findings."))
	privPaper := &Blog{
		AuthorEmail: "priv@test.edu",
		Title:       "Confidential Research",
		ContentHash: hash,
		IsPrivate:   true,
		DateCreated: "2026-01-01",
	}
	paperId, err := server.storage.CreateBlog(privPaper)
	if err != nil {
		t.Fatalf("failed to create private blog: %v", err)
	}

	// Anonymous visitor trying to access private paper -> Expect error
	reqAnon := httptest.NewRequest("GET", fmt.Sprintf("/blog/%d", paperId), nil)
	rrAnon := httptest.NewRecorder()
	server.mux.ServeHTTP(rrAnon, reqAnon)

	if rrAnon.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for anonymous user, got %d", rrAnon.Code)
	}
	if !strings.Contains(rrAnon.Body.String(), "restricted access") {
		t.Fatalf("expected restricted access message")
	}

	// Admin accessing private paper -> Expect success (200 OK)
	adminToken, _ := createJWT("admin@mintspace.io", "Editorial Admin", true)
	reqAdmin := httptest.NewRequest("GET", fmt.Sprintf("/blog/%d", paperId), nil)
	reqAdmin.AddCookie(&http.Cookie{Name: "Auth", Value: adminToken})
	rrAdmin := httptest.NewRecorder()
	server.mux.ServeHTTP(rrAdmin, reqAdmin)

	if rrAdmin.Code != http.StatusOK {
		t.Fatalf("expected admin to view private paper, got %d", rrAdmin.Code)
	}
}

func TestAdminConsole(t *testing.T) {
	server := createIsolatedTestServer(t)

	// Non-admin -> 403
	userToken, _ := createJWT("scholar@mit.edu", "Scholar", false)
	reqUser := httptest.NewRequest("GET", "/admin", nil)
	reqUser.AddCookie(&http.Cookie{Name: "Auth", Value: userToken})
	rrUser := httptest.NewRecorder()
	server.mux.ServeHTTP(rrUser, reqUser)

	if rrUser.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin on /admin, got %d", rrUser.Code)
	}

	// Admin -> 200 OK
	adminToken, _ := createJWT("admin@mintspace.io", "Editorial Admin", true)
	reqAdmin := httptest.NewRequest("GET", "/admin", nil)
	reqAdmin.AddCookie(&http.Cookie{Name: "Auth", Value: adminToken})
	rrAdmin := httptest.NewRecorder()
	server.mux.ServeHTTP(rrAdmin, reqAdmin)

	if rrAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin on /admin, got %d", rrAdmin.Code)
	}
	if !strings.Contains(rrAdmin.Body.String(), "EDITORIAL BOARD CONSOLE") {
		t.Fatalf("expected editorial console heading")
	}
}

func TestHealthzEndpoint(t *testing.T) {
	server := createIsolatedTestServer(t)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d", rr.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode healthz JSON: %v", err)
	}
	if res["status"] != "healthy" {
		t.Fatalf("expected status 'healthy', got %v", res["status"])
	}
}

func TestSecurityHeaders(t *testing.T) {
	server := createIsolatedTestServer(t)
	handler := securityHeadersMiddleware(server.mux)

	req := httptest.NewRequest("GET", "/blog", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options header")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing X-Frame-Options header")
	}
	if rr.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("missing Referrer-Policy header")
	}
}

func TestPathTraversalProtection(t *testing.T) {
	server := createIsolatedTestServer(t)

	// Attempt path traversal attacks on /blob/:hash
	badHashes := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32",
		"invalid-short-hash",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", // 64 chars but non-hex
	}

	for _, bad := range badHashes {
		req := httptest.NewRequest("GET", "/blob/"+bad, nil)
		rr := httptest.NewRecorder()
		server.mux.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			t.Fatalf("expected security rejection for bad hash '%s', got 200 OK", bad)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(1, 3) // 1 token/sec, burst 3
	testIP := "192.168.1.100"

	// Initial 3 requests should be allowed (burst)
	if !limiter.Allow(testIP) || !limiter.Allow(testIP) || !limiter.Allow(testIP) {
		t.Fatalf("expected initial burst to be allowed")
	}

	// 4th request within 0ms should be rejected
	if limiter.Allow(testIP) {
		t.Fatalf("expected 4th rapid request to be throttled by rate limiter")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDistributedCASMeshSync(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	localStore1, _ := NewLocalCASStorage(tempDir1)
	localStore2, _ := NewLocalCASStorage(tempDir2)

	node1CAS := NewDistributedCASStorage(localStore1, nil)
	node2CAS := NewDistributedCASStorage(localStore2, nil)

	// Content saved on Node 1
	manuscript := []byte("# Distributed Consensus in Heterogeneous Peer Networks\n\nContent verification verified via SHA-256.")
	hash1, err := node1CAS.Save(manuscript)
	if err != nil {
		t.Fatalf("failed to save on node 1: %v", err)
	}

	// Register peer node in mesh
	mockPeerURL := "http://peer-node-1:3000"
	node2CAS.AddPeer(mockPeerURL)

	// In-memory mock transport for peer node 1 (avoids sandbox TCP socket restrictions)
	node2CAS.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.HasPrefix(req.URL.Path, "/blob/") {
				h := strings.TrimPrefix(req.URL.Path, "/blob/")
				data, err := node1CAS.Get(h)
				if err != nil {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewReader(nil)),
						Header:     make(http.Header),
					}, nil
				}
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(data)),
					Header:     make(http.Header),
				}
				resp.Header.Set("Content-Type", "text/markdown")
				return resp, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	// Node 2 does not have the blob locally yet
	if localStore2.Exists(hash1) {
		t.Fatalf("expected node 2 not to have blob initially")
	}

	// Node 2 fetches via DistributedCASStorage -> should query Node 1, verify hash, self-heal and return!
	retrieved, err := node2CAS.Get(hash1)
	if err != nil {
		t.Fatalf("node 2 failed to fetch from peer mesh: %v", err)
	}

	if string(retrieved) != string(manuscript) {
		t.Fatalf("content mismatch retrieved from peer mesh")
	}

	// Verify self-healing: blob should now be persisted in Node 2's local CAS store!
	if !localStore2.Exists(hash1) {
		t.Fatalf("expected self-healing local persistence on node 2 after peer retrieval")
	}
}
