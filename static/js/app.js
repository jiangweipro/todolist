document.addEventListener('DOMContentLoaded', () => {
    // DOM元素
    const todoList = document.getElementById('todo-list');
    const newTodoInput = document.getElementById('new-todo');
    const addBtn = document.getElementById('add-btn');
    const todoTemplate = document.getElementById('todo-item-template');
    const usernameElement = document.getElementById('username');
    const logoutBtn = document.getElementById('logout-btn');
    
    // 优先级选择器
    const prioritySelect = document.createElement('select');
    prioritySelect.id = 'priority-select';
    prioritySelect.innerHTML = `
        <option value="0">低优先级</option>
        <option value="1" selected>中优先级</option>
        <option value="2">高优先级</option>
    `;
    
    // 将优先级选择器添加到添加待办事项的区域
    const addTodoDiv = document.querySelector('.add-todo');
    addTodoDiv.insertBefore(prioritySelect, addBtn);

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
            
            // 按优先级分组
            const priorityGroups = {
                0: [],  // 低优先级
                1: [],  // 中优先级
                2: []   // 高优先级
            };
            
            // 将待办事项按优先级分组
            todos.forEach(todo => {
                const priority = todo.priority || 1;
                priorityGroups[priority].push(todo);
            });
            
            // 创建优先级区域
            const priorityTitles = ['低优先级', '中优先级', '高优先级'];
            const priorityClasses = ['priority-section-low', 'priority-section-medium', 'priority-section-high'];
            
            // 清空列表
            todoList.innerHTML = '';
            
            // 为每个优先级创建一个区域
            for (let priority = 2; priority >= 0; priority--) {
                if (priorityGroups[priority].length > 0) {
                    // 创建优先级区域
                    const prioritySection = document.createElement('div');
                    prioritySection.className = `priority-section ${priorityClasses[priority]}`;
                    prioritySection.dataset.priority = priority;
                    
                    // 添加标题
                    const sectionTitle = document.createElement('h3');
                    sectionTitle.textContent = priorityTitles[priority];
                    prioritySection.appendChild(sectionTitle);
                    
                    // 创建待办事项容器
                    const sectionItems = document.createElement('div');
                    sectionItems.className = 'priority-items';
                    sectionItems.dataset.priority = priority;
                    prioritySection.appendChild(sectionItems);
                    
                    // 添加拖放事件
                    sectionItems.addEventListener('dragover', handleDragOver);
                    sectionItems.addEventListener('drop', handleDrop);
                    
                    // 按顺序添加待办事项
                    priorityGroups[priority]
                        .sort((a, b) => (a.order || 0) - (b.order || 0))
                        .forEach(todo => {
                            const todoNode = createTodoNode(todo);
                            sectionItems.appendChild(todoNode);
                        });
                    
                    // 添加到主列表
                    todoList.appendChild(prioritySection);
                }
            }
        } catch (error) {
            console.error('加载待办事项失败:', error);
        }
    }
    
    // 创建待办事项节点
    function createTodoNode(todo) {
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
        todoItem.dataset.order = todo.order || 0;
        todoItem.dataset.priority = todo.priority || 1;
        todoTitle.textContent = todo.title;
        checkbox.checked = todo.completed;
        
        if (todo.completed) {
            todoItem.classList.add('completed');
        }
        
        // 根据优先级添加样式
        const priorityClass = ['priority-low', 'priority-medium', 'priority-high'];
        todoItem.classList.add(priorityClass[todo.priority || 1]);
        
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

        // 添加拖放功能
        todoItem.draggable = true;
        
        todoItem.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/plain', todo.id);
            todoItem.classList.add('dragging');
        });
        
        todoItem.addEventListener('dragend', () => {
            todoItem.classList.remove('dragging');
        });

        return todoItem;
    }
    
    // 处理拖动经过事件
    function handleDragOver(e) {
        e.preventDefault();
        const section = e.currentTarget;
        const afterElement = getDragAfterElement(section, e.clientY);
        const draggable = document.querySelector('.dragging');
        
        if (afterElement == null) {
            section.appendChild(draggable);
        } else {
            section.insertBefore(draggable, afterElement);
        }
    }
    
    // 获取拖动后的位置元素
    function getDragAfterElement(container, y) {
        const draggableElements = [...container.querySelectorAll('.todo-item:not(.dragging)')];
        
        return draggableElements.reduce((closest, child) => {
            const box = child.getBoundingClientRect();
            const offset = y - box.top - box.height / 2;
            
            if (offset < 0 && offset > closest.offset) {
                return { offset: offset, element: child };
            } else {
                return closest;
            }
        }, { offset: Number.NEGATIVE_INFINITY }).element;
    }
    
    // 处理放置事件
    function handleDrop(e) {
        e.preventDefault();
        const todoId = e.dataTransfer.getData('text/plain');
        const todoItem = document.querySelector(`.todo-item[data-id="${todoId}"]`);
        const prioritySection = e.currentTarget;
        const priority = parseInt(prioritySection.dataset.priority);
        
        // 计算新的顺序
        const items = Array.from(prioritySection.querySelectorAll('.todo-item'));
        const newOrder = items.indexOf(todoItem);
        
        // 更新优先级和顺序
        updateTodoOrder(todoId, newOrder, priority);
    }
    
    // 更新待办事项顺序和优先级
    async function updateTodoOrder(todoId, order, priority) {
        try {
            const response = await fetch('/api/todos/update-order', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    todo_id: parseInt(todoId),
                    order: order
                })
            });
            
            if (!response.ok) {
                throw new Error('更新待办事项顺序失败');
            }
            
            // 更新成功后，重新加载待办事项
            loadTodos();
        } catch (error) {
            console.error('更新待办事项顺序失败:', error);
        }
    }

    // 添加新待办事项
    async function addTodo() {
        const title = newTodoInput.value.trim();
        if (!title) return;

        // 获取选择的优先级
        const priority = parseInt(prioritySelect.value);

        try {
            const response = await fetch('/api/todos', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ 
                    title: title,
                    priority: priority 
                })
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
        todoItem.dataset.order = todo.order || 0;
        todoItem.dataset.priority = todo.priority || 1;
        todoTitle.textContent = todo.title;
        checkbox.checked = todo.completed;
        
        if (todo.completed) {
            todoItem.classList.add('completed');
        }
        
        // 根据优先级添加样式
        const priorityClass = ['priority-low', 'priority-medium', 'priority-high'];
        todoItem.classList.add(priorityClass[todo.priority || 1]);
        
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

        // 添加拖放功能
        todoItem.draggable = true;
        
        todoItem.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('text/plain', todo.id);
            todoItem.classList.add('dragging');
        });
        
        todoItem.addEventListener('dragend', () => {
            todoItem.classList.remove('dragging');
        });

        // 添加到列表
        todoList.appendChild(todoNode);
    }
});