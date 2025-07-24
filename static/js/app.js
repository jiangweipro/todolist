document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const todoList = document.getElementById('todo-list');
    const newTodoInput = document.getElementById('new-todo');
    const addBtn = document.getElementById('add-btn');
    const todoTemplate = document.getElementById('todo-item-template');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');

    // 获取当前用户信息
    getCurrentUser();
    
    // 加载所有待办事项
    loadTodos();
    
    // 登出按钮事件监听
    if (logoutBtn) {
        logoutBtn.addEventListener('click', logout);
    }

    // 添加新待办事项的事件监听
    addBtn.addEventListener('click', addTodo);
    newTodoInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            addTodo();
        }
    });

    // 当前用户信息
    let currentUser = null;
    
    // 获取当前用户信息
    async function getCurrentUser() {
        try {
            const response = await fetch('/api/current-user');
            if (response.ok) {
                const userData = await response.json();
                // 保存当前用户信息
                currentUser = userData;
                
                if (usernameElement) {
                    usernameElement.textContent = userData.username;
                }
                
                // 显示管理员标识
                const adminBadge = document.getElementById('admin-badge');
                if (adminBadge && userData.is_admin) {
                    adminBadge.style.display = 'inline-block';
                }
            } else {
                // 如果未登录，重定向到登录页面
                window.location.href = '/login';
            }
        } catch (error) {
            console.error('获取用户信息失败:', error);
            window.location.href = '/login';
        }
    }
    
    // 登出功能
    async function logout() {
        try {
            const response = await fetch('/logout', {
                method: 'POST'
            });
            
            if (response.ok) {
                window.location.href = '/login';
            }
        } catch (error) {
            console.error('登出失败:', error);
        }
    }
    
    // 加载所有待办事项
    async function loadTodos() {
        try {
            const response = await fetch('/api/todos');
            const todos = await response.json();
            
            // 清空列表
            todoList.innerHTML = '';
            
            // 添加所有待办事项到列表
            todos.forEach(todo => {
                appendTodoToDOM(todo);
            });
        } catch (error) {
            console.error('加载待办事项失败:', error);
        }
    }

    // 添加新待办事项
    async function addTodo() {
        const title = newTodoInput.value.trim();
        if (!title) return;

        try {
            const response = await fetch('/api/todos', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ title })
            });

            const newTodo = await response.json();
            appendTodoToDOM(newTodo);
            
            // 清空输入框
            newTodoInput.value = '';
        } catch (error) {
            console.error('添加待办事项失败:', error);
        }
    }

    // 切换待办事项状态
    async function toggleTodo(id, checkbox) {
        try {
            const response = await fetch(`/api/todos/toggle/${id}`, {
                method: 'POST'
            });

            if (response.ok) {
                const todoItem = checkbox.closest('.todo-item');
                todoItem.classList.toggle('completed');
            }
        } catch (error) {
            console.error('切换待办事项状态失败:', error);
            // 恢复复选框状态
            checkbox.checked = !checkbox.checked;
        }
    }

    // 标记待办事项为已删除
async function markTodoAsDeleted(id, todoElement) {
    try {
        const response = await fetch(`/api/todos/mark-deleted/${id}`, {
            method: 'DELETE'
        });

            if (response.ok) {
                todoElement.remove();
            }
        } catch (error) {
            console.error('标记待办事项为已删除失败:', error);
        }
    }

    // 将待办事项添加到DOM
    function appendTodoToDOM(todo) {
        // 克隆模板
        const todoNode = document.importNode(todoTemplate.content, true);
        const todoItem = todoNode.querySelector('.todo-item');
        const checkbox = todoNode.querySelector('.todo-checkbox');
        const todoTitle = todoNode.querySelector('.todo-title');
        const todoUser = todoNode.querySelector('.todo-user');
        const deleteBtn = todoNode.querySelector('.delete-btn');

        // 设置数据
        todoItem.dataset.id = todo.id;
        todoItem.dataset.userId = todo.user_id;
        todoTitle.textContent = todo.title;
        checkbox.checked = todo.completed;
        
        if (todo.completed) {
            todoItem.classList.add('completed');
        }
        
        // 如果是管理员且待办事项不是当前用户的，显示用户名
        if (currentUser && currentUser.is_admin && todo.user_id !== currentUser.id) {
            todoUser.textContent = todo.username || `用户 ${todo.user_id}`;
            todoUser.style.display = 'inline-block';
            // 为其他用户的待办事项添加特殊样式
            todoItem.style.borderLeft = '3px solid #3498db';
        }

        // 添加事件监听
        checkbox.addEventListener('change', () => {
            toggleTodo(todo.id, checkbox);
        });

        deleteBtn.addEventListener('click', () => {
            markTodoAsDeleted(todo.id, todoItem);
        });

        // 添加到列表
        todoList.appendChild(todoNode);
    }
});