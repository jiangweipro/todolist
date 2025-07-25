package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// User 表示一个用户
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"` // 实际应用中应该存储密码哈希
	IsAdmin  bool   `json:"is_admin"` // 是否为管理员
}

// Session 表示用户会话
type Session struct {
	Token     string    `json:"token"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Todo 表示一个待办事项
type Todo struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"` // 添加用户名字段，方便前端显示
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Deleted   bool   `json:"deleted"`   // 标记待办事项是否已被删除（进入已完成状态）
}

// UserStore 管理用户的存储
type UserStore struct {
	mu       sync.Mutex
	users    []User
	nextID   int
	sessions map[string]Session
}

// NewUserStore 创建一个新的UserStore
func NewUserStore() *UserStore {
	store := &UserStore{
		users:    make([]User, 0),
		nextID:   1,
		sessions: make(map[string]Session),
	}
	
	// 尝试从文件加载数据
	err := store.LoadFromFile()
	if err != nil {
		log.Printf("加载用户数据失败: %v，将使用默认数据", err)
		
		// 创建默认的admin用户
		admin := User{
			ID:       store.nextID,
			Username: "admin",
			Password: "admin", // 实际应用中应该使用安全的密码
			IsAdmin:  true,
		}
		
		store.users = append(store.users, admin)
		store.nextID++
	}
	
	return store
}

// Register 注册新用户
func (s *UserStore) Register(username, password string, isAdmin bool) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查用户名是否已存在
	for _, user := range s.users {
		if user.Username == username {
			return User{}, fmt.Errorf("用户名已存在")
		}
	}

	// 创建新用户
	user := User{
		ID:       s.nextID,
		Username: username,
		Password: password, // 实际应用中应该存储密码哈希
		IsAdmin:  isAdmin,
	}

	s.users = append(s.users, user)
	s.nextID++

	// 保存数据到文件
	go s.SaveToFile()

	return user, nil
}

// Login 用户登录
func (s *UserStore) Login(username, password string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找用户
	for _, user := range s.users {
		if user.Username == username && user.Password == password {
			// 生成会话令牌
			token, err := generateToken()
			if err != nil {
				return Session{}, err
			}

			// 创建会话
			session := Session{
				Token:     token,
				UserID:    user.ID,
				Username:  user.Username,
				IsAdmin:   user.IsAdmin,
				ExpiresAt: time.Now().Add(24 * time.Hour), // 会话有效期24小时
			}

			// 存储会话
			s.sessions[token] = session

			return session, nil
		}
	}

	return Session{}, fmt.Errorf("用户名或密码错误")
}

// GetSession 获取会话信息
func (s *UserStore) GetSession(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[token]
	if !exists || time.Now().After(session.ExpiresAt) {
		return Session{}, false
	}

	return session, true
}

// Logout 用户登出
func (s *UserStore) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
}

// TodoStore 管理待办事项的存储
type TodoStore struct {
	mu     sync.Mutex
	todos  []Todo
	nextID int
}

// NewTodoStore 创建一个新的TodoStore
func NewTodoStore() *TodoStore {
	store := &TodoStore{
		todos:  make([]Todo, 0),
		nextID: 1,
	}
	
	// 尝试从文件加载数据
	err := store.LoadFromFile()
	if err != nil {
		log.Printf("加载待办事项数据失败: %v，将使用默认数据", err)
	}
	
	return store
}

// GetAllByUserID 返回指定用户的所有待办事项
func (s *TodoStore) GetAllByUserID(userID int, includeDeleted bool) []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 获取用户名
	username := getUsernameByID(userID)
	
	userTodos := make([]Todo, 0)
	for _, todo := range s.todos {
		if todo.UserID == userID {
			// 根据includeDeleted参数决定是否包含已删除的待办事项
			if !includeDeleted && todo.Deleted {
				continue
			}
			
			// 确保待办事项有用户名
			todoCopy := todo
			if todoCopy.Username == "" {
				todoCopy.Username = username
			}
			userTodos = append(userTodos, todoCopy)
		}
	}
	
	return userTodos
}

// GetAllTodos 返回所有待办事项，用于管理员
func (s *TodoStore) GetAllTodos(includeDeleted bool) []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 返回所有待办事项的副本，并确保每个待办事项都有用户名
	allTodos := make([]Todo, 0, len(s.todos))
	
	for _, todo := range s.todos {
		// 根据includeDeleted参数决定是否包含已删除的待办事项
		if !includeDeleted && todo.Deleted {
			continue
		}
		
		// 创建副本并确保有用户名
		todoCopy := todo
		if todoCopy.Username == "" {
			todoCopy.Username = getUsernameByID(todo.UserID)
		}
		
		allTodos = append(allTodos, todoCopy)
	}
	
	return allTodos
}

// Add 添加一个新的待办事项
func (s *TodoStore) Add(userID int, title string) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取用户名
	username := getUsernameByID(userID)

	todo := Todo{
		ID:        s.nextID,
		UserID:    userID,
		Username:  username,
		Title:     title,
		Completed: false,
	}

	s.todos = append(s.todos, todo)
	s.nextID++

	// 保存数据到文件
	go s.SaveToFile()

	return todo
}

// Toggle 切换待办事项的完成状态
func (s *TodoStore) Toggle(id int, userID int, isAdmin bool) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, todo := range s.todos {
		// 如果是管理员，可以操作任何待办事项
		// 如果不是管理员，只能操作自己的待办事项
		if todo.ID == id && (isAdmin || todo.UserID == userID) {
			s.todos[i].Completed = !s.todos[i].Completed
			
			// 保存数据到文件
			go s.SaveToFile()
			
			return s.todos[i], nil
		}
	}

	return Todo{}, fmt.Errorf("todo with ID %d not found or not owned by user", id)
}

// MarkAsDeleted 将待办事项标记为已删除（进入已完成状态）
func (s *TodoStore) MarkAsDeleted(id int, userID int, isAdmin bool) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, todo := range s.todos {
		// 如果是管理员，可以操作任何待办事项
		// 如果不是管理员，只能操作自己的待办事项
		if todo.ID == id && (isAdmin || todo.UserID == userID) {
			// 标记为已删除
			s.todos[i].Deleted = true
			// 同时标记为已完成
			s.todos[i].Completed = true
			
			// 保存数据到文件
			go s.SaveToFile()
			
			return s.todos[i], nil
		}
	}

	return Todo{}, fmt.Errorf("todo with ID %d not found or not owned by user", id)
}

// Delete 永久删除一个待办事项
func (s *TodoStore) Delete(id int, userID int, isAdmin bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, todo := range s.todos {
		// 如果是管理员，可以操作任何待办事项
		// 如果不是管理员，只能操作自己的待办事项
		if todo.ID == id && (isAdmin || todo.UserID == userID) {
			// 只有已标记为删除的待办事项才能被永久删除
			if !todo.Deleted {
				return fmt.Errorf("todo with ID %d must be marked as deleted first", id)
			}
			
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			
			// 保存数据到文件
			go s.SaveToFile()
			
			return nil
		}
	}

	return fmt.Errorf("todo with ID %d not found or not owned by user", id)
}

// 获取用户名通过用户ID
func getUsernameByID(userID int) string {
	userStore.mu.Lock()
	defer userStore.mu.Unlock()
	
	for _, user := range userStore.users {
		if user.ID == userID {
			return user.Username
		}
	}
	
	return "未知用户"
}

// 生成随机令牌
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Blog 表示一篇博客
type Blog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	IsPrivate bool      `json:"is_private"` // 是否为私有博客
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Comments  []Comment `json:"comments"`
}

// Comment 表示博客评论
type Comment struct {
	ID        int       `json:"id"`
	BlogID    int       `json:"blog_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// BlogStore 管理博客的存储
type BlogStore struct {
	mu       sync.Mutex
	blogs    []Blog
	nextID   int
	nextCommentID int
}

// NewBlogStore 创建一个新的BlogStore
func NewBlogStore() *BlogStore {
	store := &BlogStore{
		blogs:    make([]Blog, 0),
		nextID:   1,
		nextCommentID: 1,
	}
	
	// 尝试从文件加载数据
	err := store.LoadFromFile()
	if err != nil {
		log.Printf("加载博客数据失败: %v，将使用默认数据", err)
	}
	
	return store
}

// SaveToFile 保存博客数据到文件
func (s *BlogStore) SaveToFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 创建要保存的数据结构
	data := struct {
		Blogs    []Blog `json:"blogs"`
		NextID   int    `json:"next_id"`
		NextCommentID int `json:"next_comment_id"`
	}{s.blogs, s.nextID, s.nextCommentID}

	// 将数据编码为JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile("data/blogs.json", jsonData, 0644)
}

// LoadFromFile 从文件加载博客数据
func (s *BlogStore) LoadFromFile() error {
	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err := os.Stat(BLOGS_FILE); os.IsNotExist(err) {
		// 文件不存在，使用默认数据
		return nil
	}

	// 读取文件
	jsonData, err := os.ReadFile("data/blogs.json")
	if err != nil {
		return err
	}

	// 解码JSON数据
	var data struct {
		Blogs    []Blog `json:"blogs"`
		NextID   int    `json:"next_id"`
		NextCommentID int `json:"next_comment_id"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	// 更新存储
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blogs = data.Blogs
	s.nextID = data.NextID
	s.nextCommentID = data.NextCommentID

	return nil
}

// GetAllBlogs 返回所有公开博客
func (s *BlogStore) GetAllBlogs() []Blog {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	publicBlogs := make([]Blog, 0)
	for _, blog := range s.blogs {
		if !blog.IsPrivate {
			// 创建副本
			blogCopy := blog
			publicBlogs = append(publicBlogs, blogCopy)
		}
	}
	
	return publicBlogs
}

// GetBlogsByUserID 返回指定用户的所有博客
func (s *BlogStore) GetBlogsByUserID(userID int, currentUserID int) []Blog {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	userBlogs := make([]Blog, 0)
	for _, blog := range s.blogs {
		// 如果是博客作者本人或者是公开博客，则可以查看
		if blog.UserID == userID && (!blog.IsPrivate || blog.UserID == currentUserID) {
			// 创建副本
			blogCopy := blog
			userBlogs = append(userBlogs, blogCopy)
		}
	}
	
	return userBlogs
}

// GetBlogByID 根据ID获取博客
func (s *BlogStore) GetBlogByID(id int, currentUserID int) (Blog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, blog := range s.blogs {
		if blog.ID == id {
			// 如果是私有博客，只有作者本人可以查看
			if blog.IsPrivate && blog.UserID != currentUserID {
				return Blog{}, fmt.Errorf("blog with ID %d is private", id)
			}
			
			// 创建副本
			blogCopy := blog
			return blogCopy, nil
		}
	}
	
	return Blog{}, fmt.Errorf("blog with ID %d not found", id)
}

// AddBlog 添加一篇新博客
func (s *BlogStore) AddBlog(userID int, title, content string, isPrivate bool) Blog {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取用户名
	username := getUsernameByID(userID)

	blog := Blog{
		ID:        s.nextID,
		UserID:    userID,
		Username:  username,
		Title:     title,
		Content:   content,
		IsPrivate: isPrivate,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Comments:  make([]Comment, 0),
	}

	s.blogs = append(s.blogs, blog)
	s.nextID++

	// 保存数据到文件
	go s.SaveToFile()

	return blog
}

// UpdateBlog 更新博客
func (s *BlogStore) UpdateBlog(id, userID int, title, content string, isPrivate bool) (Blog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, blog := range s.blogs {
		if blog.ID == id {
			// 只有作者本人可以更新博客
			if blog.UserID != userID {
				return Blog{}, fmt.Errorf("only the author can update the blog")
			}
			
			// 更新博客
			s.blogs[i].Title = title
			s.blogs[i].Content = content
			s.blogs[i].IsPrivate = isPrivate
			s.blogs[i].UpdatedAt = time.Now()
			
			// 保存数据到文件
			go s.SaveToFile()
			
			return s.blogs[i], nil
		}
	}

	return Blog{}, fmt.Errorf("blog with ID %d not found", id)
}

// DeleteBlog 删除博客
func (s *BlogStore) DeleteBlog(id, userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, blog := range s.blogs {
		if blog.ID == id {
			// 只有作者本人可以删除博客
			if blog.UserID != userID {
				return fmt.Errorf("only the author can delete the blog")
			}
			
			// 删除博客
			s.blogs = append(s.blogs[:i], s.blogs[i+1:]...)
			
			// 保存数据到文件
			go s.SaveToFile()
			
			return nil
		}
	}

	return fmt.Errorf("blog with ID %d not found", id)
}

// AddComment 添加评论
func (s *BlogStore) AddComment(blogID, userID int, content string) (Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找博客
	var blogIndex = -1
	for i, blog := range s.blogs {
		if blog.ID == blogID {
			blogIndex = i
			break
		}
	}
	
	if blogIndex == -1 {
		return Comment{}, fmt.Errorf("blog with ID %d not found", blogID)
	}
	
	// 如果是私有博客，只有作者本人可以评论
	if s.blogs[blogIndex].IsPrivate && s.blogs[blogIndex].UserID != userID {
		return Comment{}, fmt.Errorf("cannot comment on private blog")
	}

	// 获取用户名
	username := getUsernameByID(userID)

	// 创建评论
	comment := Comment{
		ID:        s.nextCommentID,
		BlogID:    blogID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// 添加评论到博客
	s.blogs[blogIndex].Comments = append(s.blogs[blogIndex].Comments, comment)
	s.nextCommentID++

	// 保存数据到文件
	go s.SaveToFile()

	return comment, nil
}

// DeleteComment 删除评论
func (s *BlogStore) DeleteComment(blogID, commentID, userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找博客
	var blogIndex = -1
	for i, blog := range s.blogs {
		if blog.ID == blogID {
			blogIndex = i
			break
		}
	}
	
	if blogIndex == -1 {
		return fmt.Errorf("blog with ID %d not found", blogID)
	}
	
	// 查找评论
	var commentIndex = -1
	for i, comment := range s.blogs[blogIndex].Comments {
		if comment.ID == commentID {
			commentIndex = i
			break
		}
	}
	
	if commentIndex == -1 {
		return fmt.Errorf("comment with ID %d not found", commentID)
	}
	
	// 只有评论作者或博客作者可以删除评论
	comment := s.blogs[blogIndex].Comments[commentIndex]
	if comment.UserID != userID && s.blogs[blogIndex].UserID != userID {
		return fmt.Errorf("only the comment author or blog author can delete the comment")
	}
	
	// 删除评论
	s.blogs[blogIndex].Comments = append(
		s.blogs[blogIndex].Comments[:commentIndex], 
		s.blogs[blogIndex].Comments[commentIndex+1:]...
	)
	
	// 保存数据到文件
	go s.SaveToFile()
	
	return nil
}

var (
	userStore = NewUserStore()
	todoStore = NewTodoStore()
	blogStore = NewBlogStore()
	templates = template.Must(template.ParseGlob("templates/*.html"))
)

// 中间件：检查用户是否已登录
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从Cookie中获取会话令牌
		cookie, err := r.Cookie("session_token")
		if err != nil {
			// 未找到会话令牌，重定向到登录页面
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// 验证会话令牌
		session, valid := userStore.GetSession(cookie.Value)
		if !valid {
			// 会话无效，重定向到登录页面
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// 将用户信息存储在请求上下文中
		r.Header.Set("X-User-ID", strconv.Itoa(session.UserID))
		r.Header.Set("X-Username", session.Username)
		r.Header.Set("X-Is-Admin", strconv.FormatBool(session.IsAdmin))

		// 调用下一个处理函数
		next(w, r)
	}
}

// 获取当前用户ID
func getCurrentUserID(r *http.Request) (int, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return 0, fmt.Errorf("未找到用户ID")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// 添加API路由获取当前用户信息
func handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	username := r.Header.Get("X-Username")
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"username": username,
		"is_admin": isAdmin,
	})
}

func main() {
	// 设置优雅关闭
	quit := make(chan struct{})
	var wg sync.WaitGroup
	
	// 启动自动保存
	startAutoSave(&wg, quit)
	
	// 捕获系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n正在关闭服务器...")
		
		// 停止自动保存
		close(quit)
		
		// 等待所有goroutine完成
		wg.Wait()
		
		// 保存数据
		fmt.Println("正在保存数据...")
		if err := userStore.SaveToFile(); err != nil {
			log.Printf("保存用户数据失败: %v\n", err)
		}
		if err := todoStore.SaveToFile(); err != nil {
			log.Printf("保存待办事项数据失败: %v\n", err)
		}
		if err := blogStore.SaveToFile(); err != nil {
			log.Printf("保存博客数据失败: %v\n", err)
		}
		
		fmt.Println("服务器已安全关闭")
		os.Exit(0)
	}()
	
	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 用户相关路由
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/logout", handleLogout)

	// 待办事项 API 路由（需要认证）
	http.HandleFunc("/api/current-user", authMiddleware(handleCurrentUser))
	http.HandleFunc("/api/todos", authMiddleware(handleTodos))
	http.HandleFunc("/api/completed-todos", authMiddleware(handleCompletedTodos))
	http.HandleFunc("/api/todos/toggle/", authMiddleware(handleToggleTodo))
	http.HandleFunc("/api/todos/mark-deleted/", authMiddleware(handleMarkTodoAsDeleted))
	http.HandleFunc("/api/todos/delete/", authMiddleware(handleDeleteTodo))

	// 博客 API 路由（需要认证）
	http.HandleFunc("/api/blogs", authMiddleware(handleBlogs))
	http.HandleFunc("/api/blogs/", authMiddleware(handleBlog))
	http.HandleFunc("/api/blogs/user/", authMiddleware(handleUserBlogs))
	http.HandleFunc("/api/blogs/comments/", authMiddleware(handleBlogComments))

	// 页面路由
	http.HandleFunc("/", authMiddleware(handleIndex))
	http.HandleFunc("/blogs", authMiddleware(handleBlogsPage))
	http.HandleFunc("/blogs/", authMiddleware(handleBlogPage))
	http.HandleFunc("/blogs/new", authMiddleware(handleNewBlogPage))
	http.HandleFunc("/blogs/edit/", authMiddleware(handleEditBlogPage))

	// 启动服务器
	fmt.Println("服务器启动在 http://localhost:8080")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户名
	username := r.Header.Get("X-Username")
	
	// 传递用户名到模板
	data := map[string]interface{}{
		"Username": username,
	}
	
	err := templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 检查用户是否为管理员
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	// 默认不包含已删除的待办事项
	includeDeleted := false
	
	// 检查是否请求包含已删除的待办事项
	includeDeletedStr := r.URL.Query().Get("include_deleted")
	if includeDeletedStr != "" {
		includeDeleted, _ = strconv.ParseBool(includeDeletedStr)
	}

	switch r.Method {
	case http.MethodGet:
		// 如果是管理员，获取所有用户的待办事项
		if isAdmin {
			json.NewEncoder(w).Encode(todoStore.GetAllTodos(includeDeleted))
		} else {
			// 否则只获取当前用户的待办事项
			json.NewEncoder(w).Encode(todoStore.GetAllByUserID(userID, includeDeleted))
		}

	case http.MethodPost:
		var todo struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 添加待办事项，关联到当前用户
		newTodo := todoStore.Add(userID, todo.Title)
		json.NewEncoder(w).Encode(newTodo)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理已完成待办事项的请求
func handleCompletedTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 检查用户是否为管理员
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	switch r.Method {
	case http.MethodGet:
		// 获取已删除（已完成）的待办事项
		if isAdmin {
			// 管理员可以查看所有用户的已完成待办事项
			var completedTodos []Todo
			allTodos := todoStore.GetAllTodos(true)
			for _, todo := range allTodos {
				if todo.Deleted {
					completedTodos = append(completedTodos, todo)
				}
			}
			json.NewEncoder(w).Encode(completedTodos)
		} else {
			// 普通用户只能查看自己的已完成待办事项
			var completedTodos []Todo
			userTodos := todoStore.GetAllByUserID(userID, true)
			for _, todo := range userTodos {
				if todo.Deleted {
					completedTodos = append(completedTodos, todo)
				}
			}
			json.NewEncoder(w).Encode(completedTodos)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleToggleTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 检查用户是否为管理员
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	idStr := r.URL.Path[len("/api/todos/toggle/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 切换待办事项状态，管理员可以操作所有待办事项
	todo, err := todoStore.Toggle(id, userID, isAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func handleMarkTodoAsDeleted(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 检查用户是否为管理员
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	idStr := r.URL.Path[len("/api/todos/mark-deleted/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 标记待办事项为已删除（已完成），管理员可以操作所有待办事项
	todo, err := todoStore.MarkAsDeleted(id, userID, isAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// 处理博客相关的请求
func handleBlogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// 获取所有公开博客
		json.NewEncoder(w).Encode(blogStore.GetAllBlogs())

	case http.MethodPost:
		// 添加新博客
		var blog struct {
			Title     string `json:"title"`
			Content   string `json:"content"`
			IsPrivate bool   `json:"is_private"`
		}

		if err := json.NewDecoder(r.Body).Decode(&blog); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 添加博客，关联到当前用户
		newBlog := blogStore.AddBlog(userID, blog.Title, blog.Content, blog.IsPrivate)
		json.NewEncoder(w).Encode(newBlog)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理单个博客的请求
func handleBlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取博客ID
	idStr := r.URL.Path[len("/api/blogs/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// 获取单个博客
		blog, err := blogStore.GetBlogByID(id, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(blog)

	case http.MethodPut:
		// 更新博客
		var blogUpdate struct {
			Title     string `json:"title"`
			Content   string `json:"content"`
			IsPrivate bool   `json:"is_private"`
		}

		if err := json.NewDecoder(r.Body).Decode(&blogUpdate); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 更新博客
		updatedBlog, err := blogStore.UpdateBlog(id, userID, blogUpdate.Title, blogUpdate.Content, blogUpdate.IsPrivate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(updatedBlog)

	case http.MethodDelete:
		// 删除博客
		err := blogStore.DeleteBlog(id, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理用户博客的请求
func handleUserBlogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	currentUserID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取目标用户ID
	idStr := r.URL.Path[len("/api/blogs/user/"):]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		// 获取用户的博客
		blogs := blogStore.GetBlogsByUserID(userID, currentUserID)
		json.NewEncoder(w).Encode(blogs)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理博客评论的请求
func handleBlogComments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 解析路径
	pathParts := strings.Split(r.URL.Path[len("/api/blogs/comments/"):], "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// 获取博客ID
	blogID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		http.Error(w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		// 添加评论
		var comment struct {
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 添加评论
		newComment, err := blogStore.AddComment(blogID, userID, comment.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(newComment)

	case http.MethodDelete:
		// 删除评论
		if len(pathParts) < 2 {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}

		// 获取评论ID
		commentID, err := strconv.Atoi(pathParts[1])
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}

		// 删除评论
		err = blogStore.DeleteComment(blogID, commentID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理博客页面
func handleBlogsPage(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户名
	username := r.Header.Get("X-Username")
	
	// 传递用户名到模板
	data := map[string]interface{}{
		"Username": username,
	}
	
	err := templates.ExecuteTemplate(w, "blogs.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 处理单个博客页面
func handleBlogPage(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取博客ID
	idStr := r.URL.Path[len("/blogs/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 获取博客
	blog, err := blogStore.GetBlogByID(id, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// 传递数据到模板
	data := map[string]interface{}{
		"Username": r.Header.Get("X-Username"),
		"Blog":     blog,
		"UserID":   userID,
	}
	
	err = templates.ExecuteTemplate(w, "blog.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 处理新建博客页面
func handleNewBlogPage(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户名
	username := r.Header.Get("X-Username")
	
	// 传递用户名到模板
	data := map[string]interface{}{
		"Username": username,
	}
	
	err := templates.ExecuteTemplate(w, "new_blog.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// 处理编辑博客页面
func handleEditBlogPage(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 获取博客ID
	idStr := r.URL.Path[len("/blogs/edit/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 获取博客
	blog, err := blogStore.GetBlogByID(id, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// 检查是否为博客作者
	if blog.UserID != userID {
		http.Error(w, "Only the author can edit the blog", http.StatusForbidden)
		return
	}

	// 传递数据到模板
	data := map[string]interface{}{
		"Username": r.Header.Get("X-Username"),
		"Blog":     blog,
	}
	
	err = templates.ExecuteTemplate(w, "edit-blog.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前用户ID
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 检查用户是否为管理员
	isAdminStr := r.Header.Get("X-Is-Admin")
	isAdmin, _ := strconv.ParseBool(isAdminStr)

	idStr := r.URL.Path[len("/api/todos/delete/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 永久删除待办事项，管理员可以操作所有待办事项
	err = todoStore.Delete(id, userID, isAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// 处理用户注册
func handleRegister(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 显示注册页面
		err := templates.ExecuteTemplate(w, "register.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		// 检查Content-Type，处理不同格式的请求
		contentType := r.Header.Get("Content-Type")

		var username, password string

		if contentType == "application/json" {
			// 处理JSON格式的请求
			var data struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&data); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "无效的JSON数据"})
				return
			}

			username = data.Username
			password = data.Password
		} else {
			// 处理表单格式的请求
			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			username = r.Form.Get("username")
			password = r.Form.Get("password")
		}

		if username == "" || password == "" {
			if contentType == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "用户名和密码不能为空"})
			} else {
				http.Error(w, "用户名和密码不能为空", http.StatusBadRequest)
			}
			return
		}

		_, err := userStore.Register(username, password, false) // 普通用户注册，非管理员
		if err != nil {
			if contentType == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}

		if contentType == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "注册成功"})
		} else {
			// 注册成功，重定向到登录页面
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理用户登录
func handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 显示登录页面
		err := templates.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:
		// 检查Content-Type，处理不同格式的请求
		contentType := r.Header.Get("Content-Type")

		var username, password string

		if contentType == "application/json" {
			// 处理JSON格式的请求
			var data struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&data); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "无效的JSON数据"})
				return
			}

			username = data.Username
			password = data.Password
		} else {
			// 处理表单格式的请求
			err := r.ParseForm()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			username = r.Form.Get("username")
			password = r.Form.Get("password")
		}

		session, err := userStore.Login(username, password)
		if err != nil {
			if contentType == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			} else {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			}
			return
		}

		// 设置会话Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    session.Token,
			Path:     "/",
			Expires:  session.ExpiresAt,
			HttpOnly: true,
		})

		if contentType == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "登录成功"})
		} else {
			// 登录成功，重定向到首页
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// 处理用户登出
func handleLogout(w http.ResponseWriter, r *http.Request) {
	// 从Cookie中获取会话令牌
	cookie, err := r.Cookie("session_token")
	if err == nil {
		// 删除会话
		userStore.Logout(cookie.Value)
	}

	// 清除Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// 重定向到登录页面
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}