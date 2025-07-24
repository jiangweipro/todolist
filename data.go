package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 数据文件路径
const (
	USERS_FILE = "data/users.json"
	TODOS_FILE = "data/todos.json"
	BLOGS_FILE = "data/blogs.json"
)

// 确保数据目录存在
func ensureDataDir() error {
	dataDir := filepath.Dir(USERS_FILE)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// 保存用户数据到文件
func (s *UserStore) SaveToFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 创建要保存的数据结构
	data := struct {
		Users  []User           `json:"users"`
		NextID int              `json:"next_id"`
	}{s.users, s.nextID}

	// 将数据编码为JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return ioutil.WriteFile(USERS_FILE, jsonData, 0644)
}

// 从文件加载用户数据
func (s *UserStore) LoadFromFile() error {
	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err := os.Stat(USERS_FILE); os.IsNotExist(err) {
		// 文件不存在，使用默认数据
		return nil
	}

	// 读取文件
	jsonData, err := ioutil.ReadFile(USERS_FILE)
	if err != nil {
		return err
	}

	// 解码JSON数据
	var data struct {
		Users  []User `json:"users"`
		NextID int    `json:"next_id"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	// 更新存储
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = data.Users
	s.nextID = data.NextID

	return nil
}

// 保存待办事项数据到文件
func (s *TodoStore) SaveToFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 创建要保存的数据结构
	data := struct {
		Todos  []Todo `json:"todos"`
		NextID int    `json:"next_id"`
	}{s.todos, s.nextID}

	// 将数据编码为JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return ioutil.WriteFile(TODOS_FILE, jsonData, 0644)
}

// 从文件加载待办事项数据
func (s *TodoStore) LoadFromFile() error {
	// 确保数据目录存在
	if err := ensureDataDir(); err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err := os.Stat(TODOS_FILE); os.IsNotExist(err) {
		// 文件不存在，使用默认数据
		return nil
	}

	// 读取文件
	jsonData, err := ioutil.ReadFile(TODOS_FILE)
	if err != nil {
		return err
	}

	// 解码JSON数据
	var data struct {
		Todos  []Todo `json:"todos"`
		NextID int    `json:"next_id"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	// 更新存储
	s.mu.Lock()
	defer s.mu.Unlock()

	s.todos = data.Todos
	s.nextID = data.NextID

	return nil
}

// 定期保存数据的函数
func startAutoSave(wg *sync.WaitGroup, quit chan struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Minute) // 每5分钟保存一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 保存数据
				if err := userStore.SaveToFile(); err != nil {
					log.Printf("保存用户数据失败: %v\n", err)
				}
				if err := todoStore.SaveToFile(); err != nil {
					log.Printf("保存待办事项数据失败: %v\n", err)
				}
				if err := blogStore.SaveToFile(); err != nil {
					log.Printf("保存博客数据失败: %v\n", err)
				}
			case <-quit:
				// 退出信号
				return
			}
		}
	}()
}