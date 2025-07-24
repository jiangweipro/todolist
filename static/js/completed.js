document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const completedTodoList = document.getElementById('completed-todo-list');
    const completedTodoTemplate = document.getElementById('completed-todo-item-template');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    const backBtn = document.getElementById('back-btn');

    // 获取当前用户信息
    getCurrentUser();
    
    // 加载已完成待办事项
    loadCompletedTodos();
    
    // 登出按钮事件监听
    if (logoutBtn) {
        logoutBtn.addEventListener('click', logout);
    }
    
    // 返回按钮事件监听
    if (backBtn) {
        backBtn.addEventListener('click', () => {
            window.location.href = '/';
        });
    }

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
    
    // 加载已完成待办事项
    async function loadCompletedTodos() {
        try {
            const response = await fetch('/api/completed-todos');
            const todos = await response.json();
            
            // 清空列表
            completedTodoList.innerHTML = '';
            
            // 添加所有已完成待办事项到列表
            todos.forEach(todo => {
                appendCompletedTodoToDOM(todo);
            });
        } catch (error) {
            console.error('加载已完成待办事项失败:', error);
        }
    }

    // 永久删除待办事项
    async function permanentlyDeleteTodo(id, todoElement) {
        try {
            const response = await fetch(`/api/todos/delete/${id}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                todoElement.remove();
            }
        } catch (error) {
            console.error('永久删除待办事项失败:', error);
        }
    }

    // 将已完成待办事项添加到DOM
    function appendCompletedTodoToDOM(todo) {
        // 克隆模板
        const todoNode = document.importNode(completedTodoTemplate.content, true);
        const todoItem = todoNode.querySelector('.completed-todo-item');
        const todoTitle = todoNode.querySelector('.completed-todo-title');
        const todoUser = todoNode.querySelector('.todo-user');
        const deleteBtn = todoNode.querySelector('.delete-btn');

        // 设置数据
        todoItem.dataset.id = todo.id;
        todoItem.dataset.userId = todo.user_id;
        todoTitle.textContent = todo.title;
        
        // 如果是管理员且待办事项不是当前用户的，显示用户名
        if (currentUser && currentUser.is_admin && todo.user_id !== currentUser.id) {
            todoUser.textContent = todo.username || `用户 ${todo.user_id}`;
            todoUser.style.display = 'inline-block';
            // 为其他用户的待办事项添加特殊样式
            todoItem.style.borderLeft = '3px solid #3498db';
        }

        // 添加事件监听
        deleteBtn.addEventListener('click', () => {
            permanentlyDeleteTodo(todo.id, todoItem);
        });

        // 添加到列表
        completedTodoList.appendChild(todoNode);
    }
});